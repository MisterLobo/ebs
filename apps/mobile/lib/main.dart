import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:http/http.dart' as http;
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:passkeys/authenticator.dart';
import 'package:passkeys/types.dart';
import 'firebase_options.dart';
import 'package:google_sign_in/google_sign_in.dart';
Future main() async {
  await dotenv.load(fileName: '.env');
  WidgetsFlutterBinding.ensureInitialized();
  await Firebase.initializeApp(
    options: DefaultFirebaseOptions.currentPlatform,
  );
  var appEnv = dotenv.env['APP_ENV'] ?? '';
  if (appEnv == 'local') {
    // await FirebaseAuth.instance.useAuthEmulator('localhost', 9099);
  }
  runApp(const MyApp());
}

class AuthObserver extends BlocObserver {
  const AuthObserver();

  @override
  void onChange(BlocBase<dynamic> bloc, Change<dynamic> change) {
    super.onChange(bloc, change);
  }
}

class AppState {
  AppState({this.token, this.code, this.status = 'ready'});
  final String? token;
  final String? code;
  final String? status;
}

class AppCubit extends Cubit<AppState> {
  AppCubit() : super(AppState(status: 'ready'));

  updateToken(String? token) => emit(AppState(token: token, code: state.code, status: state.status));
  updateCode(String? code) => emit(AppState(token: state.token, code: code, status: state.status));
  updateState(String? value) => emit(AppState(token: state.token, code: state.code, status: state.status));
  update(AppState value) => emit(value);
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  // This widget is the root of your application.
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'EBS Scanner',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.deepPurple),
      ),
      debugShowCheckedModeBanner: false,
      home: BlocProvider(
        create: (context) => AppCubit(),
        child: const MyHomePage(title: 'EBS Scanner'),
      ),
    );
  }
}

class MyHomePage extends StatefulWidget {
  const MyHomePage({super.key, required this.title});

  final String title;

  @override
  State<MyHomePage> createState() => _MyHomePageState();
}

class LocalHttpOverrides extends HttpOverrides {
  @override
  HttpClient createHttpClient(SecurityContext? context) {
    return super.createHttpClient(context)
      ..badCertificateCallback = (X509Certificate cert, String host, int port) {
        return true;
      };
  }
}

typedef AuthenticatorResponse = Map<String, Object?>;
class Authenticator {
  Future<UserCredential> signInWithGoogle() async {
    final GoogleSignInAccount? googleUser = await GoogleSignIn().signIn();

    final GoogleSignInAuthentication? googleAuth = await googleUser?.authentication;

    final credential = GoogleAuthProvider.credential(
      accessToken: googleAuth?.accessToken,
      idToken: googleAuth?.idToken,
    );

    return await FirebaseAuth.instance.signInWithCredential(credential);
  }

  Future<void> login(BuildContext context) async {
    final cred = await signInWithGoogle();
    final email = cred.user?.email ?? '';
    var apiHost = dotenv.env['API_HOST'] ?? '';
    var appEnv = dotenv.env['APP_ENV'] ?? '';
    final idToken = await cred.user?.getIdToken();
    debugPrint('idToken: $idToken');
    if (appEnv == 'local') {
      HttpOverrides.global = LocalHttpOverrides();
    }
    final response = await http.post(
      Uri.https(apiHost, '/api/v1/auth/login'),
      headers: <String, String>{
        'Authorization': '$idToken',
        'origin': 'app:mobile',
      },
      body: jsonEncode(<String, String>{
        'email': email,
      }),
    );
    if (!context.mounted) {
      return;
    }
    if (response.statusCode == 401) {
      final authMfa = response.headers['x-authenticate-mfa'];
      if (authMfa == 'true') {
        final flowId = response.headers['x-mfa-flow-id'];
        final challenge = response.headers['x-mfa-challenge'];
        debugPrint('flowId: $flowId; challenge: $challenge');
        final startResponse = await http.post(
          Uri.https(apiHost, '/api/v1/passkey/login/start'),
          headers: <String, String>{
            'authorization': '$idToken',
            'content-type': 'application/json',
            'x-mfa-flow-id': '$flowId',
            'x-authenticate-mfa': 'true',
            'x-mfa-challenge': '$challenge',
          },
          body: jsonEncode(<String, String>{
            'email': email,
          }),
        );
        if (startResponse.statusCode == 200) {
          final responseBody = jsonDecode(startResponse.body) as Map<String, dynamic>;
          debugPrint('$responseBody');
          final authenticator = PasskeyAuthenticator(debugMode: appEnv == 'local');
          final publicKey = responseBody['publicKey'];
          final art = AuthenticateRequestType(
            relyingPartyId: publicKey['rpId'],
            challenge: publicKey['challenge'],
            mediation: MediationType.Optional,
            preferImmediatelyAvailableCredentials: true,
            allowCredentials: List.from(publicKey['allowCredentials']).map((el) {
              CredentialType? ct = CredentialType(type: el['type'], id: el['id'], transports: List.from(el['transports']));
              return ct;
            }).toList(),
          );
          final platformResponse = await authenticator.authenticate(art);
          debugPrint('$platformResponse');
        }
      }
      debugPrint('authMfa: $authMfa');
    }
    if (!context.mounted) {
      return;
    }

    debugPrint('response: ${response.body}');
    if (response.statusCode == 200) {
      var responseBody = jsonDecode(response.body) as Map<String, dynamic>;
      var token = responseBody['token'];
      context.read<AppCubit>().updateToken(token);
    } else {
      debugPrint('Could not retrieve authentication token: reason: ${response.reasonPhrase} (status ${response.statusCode})');
    }
  }

  Future<AuthenticatorResponse?> verifyCode(AppState state) async {
    var apiHost = dotenv.env['API_HOST'] ?? '';
    var appEnv = dotenv.env['APP_ENV'] ?? '';
    if (appEnv == 'local') {
      HttpOverrides.global = LocalHttpOverrides();
    }
    final response = await http.post(
      Uri.https(apiHost, '/api/v1/admission'),
      headers: <String, String>{
        'Content-Type': 'application/json; charset=UTF-8',
        'Authorization': 'Bearer ${state.token ?? 'token'}',
        'origin': 'app:mobile',
      },
      body: jsonEncode(<String, String>{
        'code': state.code!,
      }),
    );

    var verifyOk = response.statusCode == 200;
    final resp = AuthenticatorResponse();
    if (!verifyOk) {
      debugPrint('error status: ${response.statusCode}');
      try {
        var responseBody = jsonDecode(response.body) as Map<String, dynamic>;
        var error = responseBody['error'];
        debugPrint('Error response from Server: $error');
        resp.addAll({'ok': false, 'status': response.statusCode, 'error': error});
        return resp;
      } catch (e) {
        debugPrint('error: $e');
        resp.addAll({'ok': false, 'status': response.statusCode});
        return resp;
      }
    }
    resp.addAll({'ok': verifyOk, 'status': response.statusCode});
    return resp;
  }

}

class _MyHomePageState extends State<MyHomePage> {
  late bool ready;
  late bool verified;
  late bool loading;
  late int status;
  bool shouldStartManually = false;

  Authenticator? authenticator;
  MobileScannerController? controller;

  MobileScannerController initController() => MobileScannerController(
    autoStart: false,
    cameraResolution: const Size(1920, 1080),
    detectionSpeed: DetectionSpeed.unrestricted,
    detectionTimeoutMs: 1000,
    autoZoom: true,
    invertImage: false,
    returnImage: false,
  );

  Future<void> verifyCode(BuildContext context, AppState state) async {
    if (!loading) return;
    final AuthenticatorResponse? response = await authenticator?.verifyCode(state);
    if (!context.mounted) {
      return;
    }
    debugPrint('$response');
    bool? ok = response?['ok'] as bool;
    int? statusCode = response?['status'] as int;
    if (!ok) {
      String? error = response?['error'] as String;
      debugPrint('Error response from Server: $error');
      setState(() {
        status = statusCode;
        loading = false;
        verified = false;
      });
    }
    context.read<AppCubit>().updateState('ready');
    setState(() {
      status = statusCode;
      loading = false;
      verified = ok;
    });
  }

  @override
  void initState() {
    super.initState();
    authenticator = Authenticator();
    FirebaseAuth.instance
        .authStateChanges()
        .listen((User? user) {
      if (user == null) {
        print('User is currently signed out!');
      } else {
        print('User is signed in!');
      }
    });
    controller = initController();
    ready = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      controller!.start();
    });
  }

  @override
  Widget build(BuildContext context) {
    late final scanWindow = Rect.fromCenter(
      center: MediaQuery.sizeOf(context).center(const Offset(0, -200)),
      width: 200,
      height: 200,
    );
    return Scaffold(
      appBar: AppBar(
        backgroundColor: Theme.of(context).colorScheme.inversePrimary,
        title: Text(widget.title),
      ),
      body: BlocConsumer<AppCubit, AppState>(
        listenWhen: (context, state) => ready,
        listener: (context, state) {
          if (state.code != null) {
            setState(() {
              ready = false;
              loading = true;
            });
            controller!.pause();
            verifyCode(context, state);
          }
        },
        builder: (context, state) {
          if (ready) {
            controller!.start();
            return Center(
              child: controller == null ? const Placeholder() : Stack(
                children: [
                  MobileScanner(
                    scanWindow: scanWindow,
                    controller: controller,
                    fit: BoxFit.contain,
                    onDetect: (capture) {
                      final scannedBarcodes = capture.barcodes;
                      final String values = scannedBarcodes
                          .map((e) => e.displayValue)
                          .join('\n');

                      context.read<AppCubit>().updateCode(values);
                    },
                  ),
                  BarcodeOverlay(
                    boxFit: BoxFit.contain,
                    controller: controller!,
                  ),
                  ScanWindowOverlay(
                    controller: controller!,
                    scanWindow: scanWindow,
                  ),
                  Align(
                    alignment: Alignment.bottomCenter,
                    child: Container(
                      alignment: Alignment.bottomCenter,
                      height: 200,
                      color: const Color.fromRGBO(0, 0, 0, 0.4),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Expanded(child: ScannedBarcodeLabel(barcodes: controller!.barcodes)),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            );
          }
          if (loading) {
            return Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  CircularProgressIndicator(),
                  Text('verifying code'),
                  ElevatedButton(
                    onPressed: () {
                      context.read<AppCubit>().updateCode(null);
                      setState(() {
                        ready = true;
                        loading = false;
                      });
                    },
                    child: Text('Abort', style: TextStyle(color: Colors.red)),
                  ),
                ],
              ),
            );
          }
          return Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.center,
              children: verified ? [
                Text('Success', style: TextStyle(fontSize: 40, color: Colors.green)),
                ElevatedButton(
                  onPressed: () {
                    context.read<AppCubit>().updateCode(null);
                    setState(() {
                      loading = false;
                      ready = true;
                    });
                  },
                  child: Text('New scan'),
                ),
              ] : [
                Text('Fail (code $status)', style: TextStyle(fontSize: 40, color: Colors.red)),
                ElevatedButton(
                  onPressed: () {
                    context.read<AppCubit>().updateCode(null);
                    setState(() {
                      loading = false;
                      ready = true;
                    });
                  },
                  child: Text('New scan', style: TextStyle(fontSize: 32)),
                ),
              ],
            ),
          );
        }
      ),
      drawer: Drawer(
        child: ListView(
          padding: EdgeInsets.zero,
          children: [
            DrawerHeader(
              decoration: BoxDecoration(color: Colors.blue),
              child: Text('Scanner', style: TextStyle(fontSize: 40, color: Colors.white)),
            ),
            ListTile(
              title: Text('Log in'),
              onTap: () {
                authenticator?.login(context);
              },
            ),
          ],
        ),
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () {
          authenticator?.login(context);
        },
        tooltip: 'Log in',
        child: const Icon(Icons.lock),
      ),
    );
  }
}

class ScannedBarcodeLabel extends StatelessWidget {
  const ScannedBarcodeLabel({required this.barcodes, super.key});

  final Stream<BarcodeCapture> barcodes;

  @override
  Widget build(BuildContext context) {
    return StreamBuilder(
      stream: barcodes,
      builder: (context, snapshot) {
        final List<Barcode> scannedBarcodes = snapshot.data?.barcodes ?? [];

        final String values = scannedBarcodes
            .map((e) => e.displayValue)
            .join('\n');

        if (scannedBarcodes.isEmpty) {
          return const Text(
            'ready',
            overflow: TextOverflow.fade,
            style: TextStyle(color: Colors.white),
          );
        }

        // context.read<AppCubit>().updateCode(values);

        return Text(
          values.isEmpty ? 'No display value' : values,
          overflow: TextOverflow.fade,
          style: const TextStyle(color: Colors.white),
        );
      },
    );
  }
}
