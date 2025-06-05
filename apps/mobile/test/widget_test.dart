// This is a basic Flutter widget test.
//
// To perform an interaction with a widget in your test, use the WidgetTester
// utility in the flutter_test package. For example, you can send tap and scroll
// gestures. You can also use WidgetTester to find child widgets in the widget
// tree, read text, and verify that the values of widget properties are correct.

import 'dart:async';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:firebase_core_platform_interface/test.dart';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart' show BlocConsumer, BlocProvider;
import 'package:flutter_test/flutter_test.dart';

import 'package:mobile/main.dart';

import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:firebase_core/firebase_core.dart';

import 'package:firebase_auth_mocks/firebase_auth_mocks.dart';
import 'package:google_sign_in_mocks/google_sign_in_mocks.dart';

import 'package:flutter/services.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:mobile_scanner/src/method_channel/mobile_scanner_method_channel.dart';
import 'package:mockito/mockito.dart' show resetMockitoState;
import 'package:mocktail/mocktail.dart';
import 'package:mockito/annotations.dart';
import 'package:http/http.dart' as http;

typedef Callback = void Function(MethodCall call);

final user = MockUser(
  isAnonymous: false,
  uid: 'someuid',
  email: 'user@localhost.test',
  displayName: 'Test User',
);
final auth = MockFirebaseAuth(mockUser: user);
class MockMobileScannerController extends Mock
  implements MobileScannerController {
  @override
  Widget buildCameraView() {
    return const Placeholder(
      fallbackHeight: 100,
      fallbackWidth: 100,
      color: Color(0xFF00FF00),
    );
  }
}

class MockMethodChannelMobileScanner extends MethodChannelMobileScanner {
  @override
  Future<void> stop({bool force = false}) async {
    // Do nothing instead of calling platform code
  }
}

void setupFirebaseAuthMocks([Callback? customHandlers]) {
  TestWidgetsFlutterBinding.ensureInitialized();

  setupFirebaseCoreMocks();
}

Future<T> neverEndingFuture<T>() async {
  // ignore: literal_only_boolean_expressions
  while (true) {
    await Future.delayed(const Duration(minutes: 5));
  }
}

class MockFAB extends Mock implements StatelessWidget {
  late final String? token;

  Future<UserCredential> signInWithGoogle() async {
    final googleSignIn = MockGoogleSignIn();
    final signInAccount = await googleSignIn.signIn();

    final googleAuth = await signInAccount?.authentication;
    final AuthCredential authCredential = GoogleAuthProvider.credential(
      accessToken: googleAuth?.accessToken,
      idToken: googleAuth?.idToken,
    );

    final user = MockUser(
      isAnonymous: false,
      uid: 'someuid',
      email: 'user@localhost.test',
      displayName: 'Test User',
    );
    final auth = MockFirebaseAuth(mockUser: user);

    return await auth.signInWithCredential(authCredential);
  }

  Future<void> login() async {}

  @override
  Widget build(BuildContext context) {
    return FloatingActionButton(onPressed: () {
      login();
    });
  }
  @override
  String toString({DiagnosticLevel minLevel = DiagnosticLevel.info}) {
    return super.toString();
  }
}

class _MockAuthenticator extends Mock implements Authenticator {
  @override
  Future<UserCredential> signInWithGoogle() async {
    final googleSignIn = MockGoogleSignIn();
    final signInAccount = await googleSignIn.signIn();

    final googleAuth = await signInAccount?.authentication;
    final AuthCredential authCredential = GoogleAuthProvider.credential(
      accessToken: googleAuth?.accessToken,
      idToken: googleAuth?.idToken,
    );

    final user = MockUser(
      isAnonymous: false,
      uid: 'someuid',
      email: 'user@localhost.test',
      displayName: 'Test User',
    );
    final auth = MockFirebaseAuth(mockUser: user);

    return await auth.signInWithCredential(authCredential);
  }
  @override
  Future<AuthenticatorResponse?> verifyCode(AppState state) async {
    final hasCode = state.code != null;
    final map = AuthenticatorResponse();
    map.addAll({'ok': hasCode, 'status': hasCode ? 200 : 400, 'error': null});
    return map;
  }
}
class _MockContext extends Mock implements BuildContext {}

@GenerateMocks([http.Client])
Future<void> main() async {
  setupFirebaseAuthMocks();
  TestWidgetsFlutterBinding.ensureInitialized();
  late StreamController<BarcodeCapture> barcodeStreamController;
  late MockMobileScannerController mockController;
  late _MockContext ctx = _MockContext();
  final mockAuth = _MockAuthenticator();

  setUp(() async {
    MobileScannerPlatform.instance = MockMethodChannelMobileScanner();
    mockController = MockMobileScannerController();
    barcodeStreamController = StreamController<BarcodeCapture>.broadcast();

    when(() => mockController.autoStart).thenReturn(true);
    when(
      () => mockController.barcodes,
    ).thenAnswer((_) => barcodeStreamController.stream);
    when(() => mockController.value).thenReturn(
      const MobileScannerState(
        availableCameras: 2,
        cameraDirection: CameraFacing.back,
        isInitialized: true, isStarting: false,
        isRunning: true, size: Size(1920, 1080),
        torchState: TorchState.off,
        zoomScale: 1,
        deviceOrientation: DeviceOrientation.portraitUp,
      ),
    );
    when(() => mockController.start()).thenAnswer((_) async {});
    when(() => mockController.pause()).thenAnswer((_) async {});
    when(() => mockController.stop()).thenAnswer((_) async {});
    when(() => mockController.dispose()).thenAnswer((_) async {});
    when(() => mockController.toggleTorch()).thenAnswer((_) async {});
    when(() => mockController.switchCamera()).thenAnswer((_) async {});
    when(() => mockController.updateScanWindow(any())).thenAnswer((_) async {});

    await dotenv.load(fileName: '.env');
    await Firebase.initializeApp();
  });

  tearDown(() {
    barcodeStreamController.close();
    resetMockitoState();
    resetMocktailState();
  });

  group('MyApp smoke test', () {
    late String token = '';
    setUp(() {
      when(() => mockAuth.login(ctx)).thenAnswer((_) async {
        token = 'token';
      });
    });

    testWidgets('Scanner smoke test', (WidgetTester tester) async {
      bool loginWasTapped = false;
      bool wasCalled = false;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Center(
              child: MobileScanner(
                controller: mockController,
                onDetect: (_) {
                  wasCalled = true;
                },
              ),
            ),
            floatingActionButton: FloatingActionButton(
              onPressed: () {
                loginWasTapped = true;
              },
              tooltip: 'Log in',
              child: const Icon(Icons.lock),
            ),
          ),
        ),
      );

      barcodeStreamController.add(const BarcodeCapture());
      await tester.pump();

      expect(wasCalled, true);

      // Tap the 'lock' icon and trigger a frame.
      await tester.tap(find.byIcon(Icons.lock));
      await tester.pump();

      expect(loginWasTapped, true);
      await barcodeStreamController.close();
    });
    testWidgets('Scanner should read code', (WidgetTester tester) async {
      bool loginWasTapped = false;
      bool wasCalled = false;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Center(
              child: MobileScanner(
                controller: mockController,
                onDetect: (_) {
                  wasCalled = true;
                },
              ),
            ),
            floatingActionButton: FloatingActionButton(
              onPressed: () {
                loginWasTapped = true;
              },
              tooltip: 'Log in',
              child: const Icon(Icons.lock),
            ),
          ),
        ),
      );

      barcodeStreamController.add(const BarcodeCapture());
      await tester.pump();

      expect(wasCalled, true);

      // Tap the 'lock' icon and trigger a frame.
      await tester.tap(find.byIcon(Icons.lock));
      await tester.pump();

      expect(loginWasTapped, true);
      await barcodeStreamController.close();
    });
    testWidgets('login should work', (WidgetTester tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Center(
              child: Placeholder(),
            ),
            floatingActionButton: FloatingActionButton(
              onPressed: () {
                mockAuth.login(ctx);
              },
              child: const Icon(Icons.lock),
            ),
          ),
        )
      );

      // Tap the 'lock' icon and trigger a frame.
      await tester.tap(find.byIcon(Icons.lock));
      await tester.pump();

      expect(token, 'token');
    });
    testWidgets('verifyCode should work', (WidgetTester tester) async {
      bool wasCalled = false;
      final cubit = AppCubit();
      late String barcodeValue = '';
      await tester.pumpWidget(
        MaterialApp(
          home: BlocProvider(
            create: (context) => cubit,
            child: Scaffold(
              body: BlocConsumer<AppCubit, AppState>(
                listener: (context, state) {
                  if (state.code != null) {
                    mockController.pause();
                    mockAuth.verifyCode(state);
                  }
                },
                builder: (context, state) {
                  return Center(
                    child: MobileScanner(
                      controller: mockController,
                      onDetect: (capture) {
                        wasCalled = true;

                        final scannedBarcodes = capture.barcodes;
                        final String values = scannedBarcodes
                          .map((e) => e.displayValue)
                          .join('\n');

                        barcodeValue = values;
                        cubit.updateCode(values);
                      },
                    ),
                  );
                },
              ),
            ),
          ),
        ),
      );
      barcodeStreamController.add(BarcodeCapture(
        barcodes: List.generate(1, (int index) => Barcode(
          displayValue: 'barcode',
        )),
      ));
      await tester.pump();

      expect(wasCalled, true);

      await tester.pump();
      expect(barcodeValue, 'barcode');

      await barcodeStreamController.close();
    });
  });
}
