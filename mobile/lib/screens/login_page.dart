import 'package:flutter/material.dart';

import '../services/api.dart';
import '../services/session.dart';
import 'change_password_page.dart';
import 'home_shell.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({super.key});
  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final user = TextEditingController();
  final pass = TextEditingController();
  String mode = 'local';
  String? err;
  bool loading = false;

  Future<void> login() async {
    setState(() {
      loading = true;
      err = null;
    });
    try {
      final data = await Api.post('/auth/login', {
        'username': user.text.trim(),
        'password': pass.text,
        'mode': mode,
      }, auth: false);
      final clients = (data['clients'] as List?)?.map((e) => '$e').toList() ?? [];
      if (clients.isNotEmpty && !clients.contains('app')) {
        throw Exception('当前账号无 APP 访问权');
      }
      await Session.saveLogin(data);
      if (!mounted) return;
      if (data['mustChangePassword'] == true) {
        Navigator.of(context).pushReplacement(MaterialPageRoute(builder: (_) => const ChangePasswordPage()));
      } else {
        Navigator.of(context).pushReplacement(MaterialPageRoute(builder: (_) => const HomeShell()));
      }
    } catch (e) {
      setState(() => err = e.toString());
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
            const Text('危废管控 APP', style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold)),
            const SizedBox(height: 8),
            const Text('现场执行 · 弱网可用', style: TextStyle(color: Colors.black54)),
            const SizedBox(height: 24),
            TextField(controller: user, decoration: const InputDecoration(labelText: '账号')),
            TextField(controller: pass, obscureText: true, decoration: const InputDecoration(labelText: '密码')),
            const SizedBox(height: 8),
            SegmentedButton<String>(
              segments: const [
                ButtonSegment(value: 'local', label: Text('本地')),
                ButtonSegment(value: 'ldap', label: Text('LDAP')),
              ],
              selected: {mode},
              onSelectionChanged: (s) => setState(() => mode = s.first),
            ),
            if (err != null) Padding(padding: const EdgeInsets.only(top: 8), child: Text(err!, style: const TextStyle(color: Colors.red))),
            const SizedBox(height: 16),
            SizedBox(
              width: double.infinity,
              height: 48,
              child: FilledButton(onPressed: loading ? null : login, child: const Text('登录')),
            ),
          ]),
        ),
      ),
    );
  }
}
