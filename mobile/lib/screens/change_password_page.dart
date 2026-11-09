import 'package:flutter/material.dart';

import '../services/api.dart';
import '../services/session.dart';
import 'home_shell.dart';

class ChangePasswordPage extends StatefulWidget {
  const ChangePasswordPage({super.key});
  @override
  State<ChangePasswordPage> createState() => _ChangePasswordPageState();
}

class _ChangePasswordPageState extends State<ChangePasswordPage> {
  final oldPwd = TextEditingController();
  final newPwd = TextEditingController();
  final confirm = TextEditingController();
  String? err;
  bool loading = false;

  Future<void> submit() async {
    if (newPwd.text.length < 8) {
      setState(() => err = '新密码至少 8 位');
      return;
    }
    if (newPwd.text != confirm.text) {
      setState(() => err = '两次新密码不一致');
      return;
    }
    setState(() {
      loading = true;
      err = null;
    });
    try {
      await Api.post('/auth/change-password', {
        'oldPassword': oldPwd.text,
        'newPassword': newPwd.text,
      });
      await Session.clearMustChange();
      if (!mounted) return;
      Navigator.of(context).pushReplacement(MaterialPageRoute(builder: (_) => const HomeShell()));
    } catch (e) {
      setState(() => err = e.toString());
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('强制修改密码')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(children: [
          const Text('首次登录或管理员要求改密后需设置新密码'),
          TextField(controller: oldPwd, obscureText: true, decoration: const InputDecoration(labelText: '原密码')),
          TextField(controller: newPwd, obscureText: true, decoration: const InputDecoration(labelText: '新密码')),
          TextField(controller: confirm, obscureText: true, decoration: const InputDecoration(labelText: '确认密码')),
          if (err != null) Text(err!, style: const TextStyle(color: Colors.red)),
          const SizedBox(height: 16),
          SizedBox(
            width: double.infinity,
            height: 48,
            child: FilledButton(onPressed: loading ? null : submit, child: const Text('提交')),
          ),
        ]),
      ),
    );
  }
}
