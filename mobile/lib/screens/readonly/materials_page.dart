import 'package:flutter/material.dart';

import '../../services/api.dart';
import '../scan_page.dart';

class MaterialsPage extends StatefulWidget {
  const MaterialsPage({super.key});
  @override
  State<MaterialsPage> createState() => _MaterialsPageState();
}

class _MaterialsPageState extends State<MaterialsPage> {
  List list = [];
  Map? detail;
  String? err;
  final q = TextEditingController();

  Future<void> _search([String? code]) async {
    setState(() {
      err = null;
      detail = null;
    });
    try {
      final key = (code ?? q.text).trim();
      if (key.isEmpty) {
        final data = await Api.get('/unique-codes?page=1&pageSize=30');
        list = (data is Map) ? (data['list'] as List? ?? []) : [];
      } else {
        final d = await Api.get('/unique-codes/$key');
        detail = d is Map ? Map<String, dynamic>.from(d) : null;
        list = detail == null ? [] : [detail!];
      }
      setState(() {});
    } catch (e) {
      setState(() => err = e.toString());
    }
  }

  @override
  void initState() {
    super.initState();
    _search();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('物资/唯一码')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Row(
            children: [
              Expanded(child: TextField(controller: q, decoration: const InputDecoration(labelText: '唯一码'))),
              IconButton(
                icon: const Icon(Icons.qr_code_scanner),
                onPressed: () async {
                  final v = await Navigator.of(context).push<String>(
                    MaterialPageRoute(builder: (_) => const ScanPage(title: '扫码查码')),
                  );
                  if (v != null) {
                    q.text = v;
                    _search(v);
                  }
                },
              ),
              IconButton(icon: const Icon(Icons.search), onPressed: _search),
            ],
          ),
          if (err != null) Text(err!, style: const TextStyle(color: Colors.red)),
          ...list.map((e) {
            final m = e as Map;
            return Card(
              child: ListTile(
                title: Text('${m['Code'] ?? m['Name'] ?? '-'}'),
                subtitle: Text('状态 ${m['Status'] ?? '-'} · 物资 ${m['MaterialID'] ?? '-'} · 仓 ${m['WarehouseID'] ?? '-'}'),
              ),
            );
          }),
        ],
      ),
    );
  }
}
