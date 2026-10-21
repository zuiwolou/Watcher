import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';
import 'package:uuid/uuid.dart';

const apiBase = String.fromEnvironment('API_BASE', defaultValue: 'http://10.0.2.2:8080/api/v1');

void main() {
  runApp(const WatcherApp());
}

class WatcherApp extends StatelessWidget {
  const WatcherApp({super.key});
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: '危废管控',
      theme: ThemeData(colorSchemeSeed: const Color(0xFF0B3D4A), useMaterial3: true),
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
    final sp = await SharedPreferences.getInstance();
    final token = sp.getString('accessToken');
    if (!mounted) return;
    Navigator.of(context).pushReplacement(
      MaterialPageRoute(builder: (_) => token == null ? const LoginPage() : const HomePage()),
    );
  }

  @override
  Widget build(BuildContext context) => const Scaffold(body: Center(child: CircularProgressIndicator()));
}

class Api {
  static Future<Map<String, dynamic>> post(String path, Map body, {bool auth = true}) async {
    final sp = await SharedPreferences.getInstance();
    final headers = {'Content-Type': 'application/json', 'X-Client': 'app'};
    if (auth) {
      final t = sp.getString('accessToken');
      if (t != null) headers['Authorization'] = 'Bearer $t';
    }
    final res = await http.post(Uri.parse('$apiBase$path'), headers: headers, body: jsonEncode(body));
    final data = jsonDecode(res.body) as Map<String, dynamic>;
    if (data['code'] != 'OK') {
      throw Exception(data['message'] ?? data['code']);
    }
    return data['data'] as Map<String, dynamic>? ?? {};
  }

  static Future<dynamic> get(String path) async {
    final sp = await SharedPreferences.getInstance();
    final headers = {'X-Client': 'app'};
    final t = sp.getString('accessToken');
    if (t != null) headers['Authorization'] = 'Bearer $t';
    final res = await http.get(Uri.parse('$apiBase$path'), headers: headers);
    final data = jsonDecode(res.body) as Map<String, dynamic>;
    if (data['code'] != 'OK') {
      throw Exception(data['message'] ?? data['code']);
    }
    return data['data'];
  }
}

class LoginPage extends StatefulWidget {
  const LoginPage({super.key});
  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final user = TextEditingController(text: 'admin');
  final pass = TextEditingController(text: 'Admin@123');
  String? err;
  bool loading = false;

  Future<void> login() async {
    setState(() { loading = true; err = null; });
    try {
      final data = await Api.post('/auth/login', {
        'username': user.text,
        'password': pass.text,
        'mode': 'local',
      }, auth: false);
      final sp = await SharedPreferences.getInstance();
      await sp.setString('accessToken', data['accessToken']);
      await sp.setString('refreshToken', data['refreshToken'] ?? '');
      await sp.setString('permissions', jsonEncode(data['permissions'] ?? []));
      if (!mounted) return;
      Navigator.of(context).pushReplacement(MaterialPageRoute(builder: (_) => const HomePage()));
    } catch (e) {
      setState(() => err = e.toString());
    } finally {
      setState(() => loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
          const Text('危废管控 APP', style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold)),
          const SizedBox(height: 24),
          TextField(controller: user, decoration: const InputDecoration(labelText: '账号')),
          TextField(controller: pass, obscureText: true, decoration: const InputDecoration(labelText: '密码')),
          if (err != null) Text(err!, style: const TextStyle(color: Colors.red)),
          const SizedBox(height: 16),
          FilledButton(onPressed: loading ? null : login, child: const Text('登录')),
        ]),
      ),
    );
  }
}

class HomePage extends StatefulWidget {
  const HomePage({super.key});
  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  int idx = 0;
  List perms = [];

  @override
  void initState() {
    super.initState();
    SharedPreferences.getInstance().then((sp) {
      setState(() {
        perms = jsonDecode(sp.getString('permissions') ?? '[]') as List;
      });
    });
  }

  bool has(String p) => perms.contains(p);

  @override
  Widget build(BuildContext context) {
    final tabs = <Widget>[
      const _AlarmsTab(),
      if (has('inbound:confirm')) const _ExecTab(),
      const _MineTab(),
    ];
    final items = <BottomNavigationBarItem>[
      const BottomNavigationBarItem(icon: Icon(Icons.warning_amber), label: '告警'),
      if (has('inbound:confirm')) const BottomNavigationBarItem(icon: Icon(Icons.qr_code_scanner), label: '执行'),
      const BottomNavigationBarItem(icon: Icon(Icons.person), label: '我的'),
    ];
    final safeIdx = idx.clamp(0, tabs.length - 1);
    return Scaffold(
      appBar: AppBar(title: const Text('危废管控')),
      body: tabs[safeIdx],
      bottomNavigationBar: BottomNavigationBar(
        currentIndex: safeIdx,
        onTap: (i) => setState(() => idx = i),
        items: items,
      ),
    );
  }
}

class _AlarmsTab extends StatefulWidget {
  const _AlarmsTab();
  @override
  State<_AlarmsTab> createState() => _AlarmsTabState();
}

class _AlarmsTabState extends State<_AlarmsTab> {
  List list = [];
  @override
  void initState() {
    super.initState();
    reload();
  }

  Future<void> reload() async {
    final data = await Api.get('/alarms');
    setState(() => list = (data['list'] as List?) ?? []);
  }

  @override
  Widget build(BuildContext context) {
    return RefreshIndicator(
      onRefresh: reload,
      child: ListView.builder(
        itemCount: list.length,
        itemBuilder: (_, i) {
          final a = list[i] as Map;
          return ListTile(
            title: Text('${a['Title']}'),
            subtitle: Text('${a['Status']} · ${a['Level']}'),
            trailing: a['Status'] == 'raised'
                ? TextButton(
                    child: const Text('确认'),
                    onPressed: () async {
                      await Api.post('/alarms/${a['ID']}/ack', {});
                      reload();
                    },
                  )
                : null,
          );
        },
      ),
    );
  }
}

class _ExecTab extends StatefulWidget {
  const _ExecTab();
  @override
  State<_ExecTab> createState() => _ExecTabState();
}

class _ExecTabState extends State<_ExecTab> {
  final orderId = TextEditingController();
  final code = TextEditingController();
  String msg = '';

  Future<void> confirmInbound() async {
    try {
      await Api.post('/inbound/orders/${orderId.text}/confirm', {
        'uniqueCode': code.text,
        'clientRequestId': const Uuid().v4(),
      });
      setState(() => msg = '入库确认成功');
    } catch (e) {
      setState(() => msg = e.toString());
    }
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(children: [
        const Text('入库扫码确认（可手输唯一码）', style: TextStyle(fontWeight: FontWeight.bold)),
        TextField(controller: orderId, decoration: const InputDecoration(labelText: '入库单ID')),
        TextField(controller: code, decoration: const InputDecoration(labelText: '唯一码')),
        FilledButton(onPressed: confirmInbound, child: const Text('提交确认')),
        Text(msg),
      ]),
    );
  }
}

class _MineTab extends StatelessWidget {
  const _MineTab();
  @override
  Widget build(BuildContext context) {
    return Center(
      child: FilledButton(
        onPressed: () async {
          final sp = await SharedPreferences.getInstance();
          await sp.clear();
          if (!context.mounted) return;
          Navigator.of(context).pushAndRemoveUntil(
            MaterialPageRoute(builder: (_) => const LoginPage()),
            (_) => false,
          );
        },
        child: const Text('退出登录'),
      ),
    );
  }
}
