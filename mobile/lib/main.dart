import 'package:flutter/material.dart';

import 'screens/change_password_page.dart';
import 'screens/home_shell.dart';
import 'screens/login_page.dart';
import 'services/push_service.dart';
import 'services/session.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  try {
    await PushService.instance.init();
  } catch (_) {
    // notifications optional on unsupported platforms
  }
  runApp(const WatcherApp());
}

class WatcherApp extends StatelessWidget {
  const WatcherApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: '危废管控',
      theme: ThemeData(
        colorSchemeSeed: const Color(0xFF0B3D4A),
        useMaterial3: true,
      ),
      home: const GatePage(),
    );
  }
}

class GatePage extends StatefulWidget {
  const GatePage({super.key});
  @override
  State<GatePage> createState() => _GatePageState();
}

class _GatePageState extends State<GatePage> {
  @override
  void initState() {
    super.initState();
    _boot();
  }

  Future<void> _boot() async {
    final token = await Session.accessToken();
    final must = await Session.mustChangePassword();
    if (!mounted) return;
    Widget next;
    if (token == null || token.isEmpty) {
      next = const LoginPage();
    } else if (must) {
      next = const ChangePasswordPage();
    } else {
      next = const HomeShell();
    }
    Navigator.of(context).pushReplacement(MaterialPageRoute(builder: (_) => next));
  }

  @override
  Widget build(BuildContext context) => const Scaffold(body: Center(child: CircularProgressIndicator()));
}
