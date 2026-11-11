import 'package:flutter/material.dart';

import '../../services/api.dart';
import 'exec_forms.dart';

class TransportPage extends StatefulWidget {
  const TransportPage({super.key});
  @override
  State<TransportPage> createState() => _TransportPageState();
}

class _TransportPageState extends State<TransportPage> with ExecSubmitMixin {
  List tasks = [];
  Map? selected;
  String nodeType = 'depart';
  final code = TextEditingController();
  bool loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => loading = true);
    try {
      final data = await Api.get('/transport/tasks?page=1&pageSize=50');
      final list = (data is Map) ? (data['list'] as List? ?? []) : <dynamic>[];
      tasks = list.where((e) {
        final s = '${(e as Map)['Status'] ?? ''}';
        return s != 'signed';
      }).toList();
    } catch (e) {
      setState(() {
        err = true;
        msg = e.toString();
      });
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  Future<void> _submit() async {
    final id = selected?['ID'];
    if (id == null) return;
    final needPhoto = nodeType == 'depart' || nodeType == 'arrive';
    try {
      await runSubmit(
        path: '/transport/checkins',
        body: {
          'taskId': id,
          'nodeType': nodeType,
          if (code.text.trim().isNotEmpty) 'uniqueCode': code.text.trim(),
        },
        label: '运输打卡',
        requireLocation: true,
        requirePhoto: needPhoto,
      );
      if (!err) _load();
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('运输打卡'), actions: [IconButton(icon: const Icon(Icons.refresh), onPressed: _load)]),
      body: loading
          ? const Center(child: CircularProgressIndicator())
          : selected == null
              ? ListView(
                  children: [
                    const Padding(
                      padding: EdgeInsets.all(16),
                      child: Text('我的运输任务', style: TextStyle(fontWeight: FontWeight.w600)),
                    ),
                    TaskPickerList(
                      items: tasks,
                      titleBuilder: (r) => '任务 ${r['TaskNo'] ?? r['ID']}',
                      subtitleBuilder: (r) => '状态 ${r['Status']} · 仓 ${r['WarehouseID']}',
                      onSelect: (r) => setState(() => selected = r),
                    ),
                    ExecResultBanner(msg: msg, isError: err),
                  ],
                )
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    ListTile(
                      contentPadding: EdgeInsets.zero,
                      title: Text('任务 ${selected!['TaskNo'] ?? selected!['ID']}'),
                      trailing: TextButton(onPressed: () => setState(() => selected = null), child: const Text('重选')),
                    ),
                    DropdownButtonFormField<String>(
                      initialValue: nodeType,
                      items: const [
                        DropdownMenuItem(value: 'depart', child: Text('发车')),
                        DropdownMenuItem(value: 'waypoint', child: Text('途经')),
                        DropdownMenuItem(value: 'arrive', child: Text('到达')),
                      ],
                      onChanged: (v) => setState(() => nodeType = v ?? 'depart'),
                      decoration: const InputDecoration(labelText: '节点'),
                    ),
                    TextField(
                      controller: code,
                      decoration: InputDecoration(
                        labelText: '唯一码（可选）',
                        suffixIcon: IconButton(
                          icon: const Icon(Icons.qr_code_scanner),
                          onPressed: () async {
                            final v = await openScan(context, title: '运输扫码');
                            if (v != null) setState(() => code.text = v);
                          },
                        ),
                      ),
                    ),
                    LocationPhotoBar(
                      fix: fix,
                      photoPath: photoPath,
                      onLocate: () => refreshLocation(required: true),
                      onPhoto: capturePhoto,
                      requirePhotoHint: nodeType == 'depart' || nodeType == 'arrive',
                    ),
                    const SizedBox(height: 12),
                    FilledButton(onPressed: busy ? null : _submit, child: const Text('提交打卡')),
                    ExecResultBanner(msg: msg, isError: err),
                  ],
                ),
    );
  }
}
