import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:get_it/get_it.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:http/http.dart' as http;
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

GetIt locator = GetIt.instance;

Future main() async {
  await dotenv.load(fileName: '.env');
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
      title: 'Flutter Demo',
      theme: ThemeData(
        // This is the theme of your application.
        //
        // TRY THIS: Try running your application with "flutter run". You'll see
        // the application has a purple toolbar. Then, without quitting the app,
        // try changing the seedColor in the colorScheme below to Colors.green
        // and then invoke "hot reload" (save your changes or press the "hot
        // reload" button in a Flutter-supported IDE, or press "r" if you used
        // the command line to start the app).
        //
        // Notice that the counter didn't reset back to zero; the application
        // state is not lost during the reload. To reset the state, use hot
        // restart instead.
        //
        // This works for code too, not just values: Most code changes can be
        // tested with just a hot reload.
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

class LoginPage extends StatefulWidget {
  const LoginPage({super.key});
  @override
  State<StatefulWidget> createState() => _LoginPageState();
}
class _LoginPageState extends State<LoginPage> {
  @override
  Widget build(BuildContext context) {
    // TODO: implement build
    throw UnimplementedError();
  }
}

class MyHomePage extends StatefulWidget {
  const MyHomePage({super.key, required this.title});

  // This widget is the home page of your application. It is stateful, meaning
  // that it has a State object (defined below) that contains fields that affect
  // how it looks.

  // This class is the configuration for the state. It holds the values (in this
  // case the title) provided by the parent (in this case the App widget) and
  // used by the build method of the State. Fields in a Widget subclass are
  // always marked "final".

  final String title;

  @override
  State<MyHomePage> createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  late bool ready;
  late bool verified;
  late bool loading;
  late int status;

  Future<void> login(BuildContext context) async {
    var apiHost = dotenv.env['API_HOST'] ?? '';
    var secret = dotenv.env['API_SECRET'] ?? '';
    debugPrint('$apiHost $secret');
    final response = await http.post(
      Uri.https(apiHost, '/api/v1/auth/login'),
      headers: <String, String>{
        'x-secret': secret,
        'origin': 'app:mobile',
      },
      body: jsonEncode(<String, String>{
        'email': 'algae1234567890@gmail.com',
      }),
    );

    debugPrint('response: ${response.body}');
    if (response.statusCode == 200) {
      var responseBody = jsonDecode(response.body) as Map<String, dynamic>;
      var token = responseBody['token'];
      context.read<AppCubit>().updateToken(token);
    } else {
      debugPrint('Could not retrieve authentication token (status ${response.statusCode})');
    }
  }

  MobileScannerController? controller;

  MobileScannerController initController() => MobileScannerController(
    autoStart: false,
    cameraResolution: const Size(1920, 1080),
    detectionSpeed: DetectionSpeed.normal,
    detectionTimeoutMs: 1000,
    autoZoom: true,
    invertImage: false,
    returnImage: false,
  );

  Future<void> verifyCode(BuildContext context, AppState state) async {
    if (!context.mounted) {
      return;
    }
    if (!loading) return;
    var apiHost = dotenv.env['API_HOST'] ?? '';
    var secret = dotenv.env['API_SECRET'] ?? '';
    final response = await http.post(
      Uri.https(apiHost, '/api/v1/admission'),
      headers: <String, String>{
        'Content-Type': 'application/json; charset=UTF-8',
        'Authorization': 'Bearer ${state.token}',
        'x-secret': secret,
        'origin': 'app:mobile',
      },
      body: jsonEncode(<String, String>{
        'code': state.code!,
      }),
    );

    var verifyOk = response.statusCode == 200;
    context.read<AppCubit>().updateState('ready');
    setState(() {
      status = response.statusCode;
      loading = false;
      verified = verifyOk;
    });
  }

  @override
  void initState() {
    super.initState();
    ready = true;
    controller = initController();
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
            // controller!.pause();
            verifyCode(context, state);
          }
        },
        builder: (context, state) {
          if (ready) {
            return Center(
              // Center is a layout widget. It takes a single child and positions it
              // in the middle of the parent.
              child: controller == null ? const Placeholder() : Stack(
                children: [
                  MobileScanner(
                    scanWindow: scanWindow,
                    controller: controller,
                    fit: BoxFit.contain,
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
                      // controller!.start();
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
                    // controller!.start();
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
                    // controller!.start();
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
                login(context);
              },
            ),
          ],
        ),
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () {
          login(context);
        },
        tooltip: 'Log in',
        child: const Icon(Icons.lock),
      ), // This trailing comma makes auto-formatting nicer for build methods.
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

        context.read<AppCubit>().updateCode(values);

        return Text(
          values.isEmpty ? 'No display value' : values,
          overflow: TextOverflow.fade,
          style: const TextStyle(color: Colors.white),
        );
      },
    );
  }
}
