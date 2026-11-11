import 'package:flutter/material.dart';

import '../../services/api.dart';
import 'exec_forms.dart';

class InventoryPage extends StatefulWidget {
  const InventoryPage({super.key});
  @override
  State<InventoryPage> createState() => _InventoryPageState();
}

class _InventoryPageState extends State<InventoryPage> with ExecSubmitMixin {
  List tasks = [];
  Map? selected;
  int scanned = 0;
  bool loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => loading = true);
    try {
      final data = await Api.get('/inventory/tasks?page=1&pageSize=50');
      final list = (data is Map) ? (data['list'] as List? ?? []) : <dynamic>[];
      tasks = list.where((e) {
        final s = '${(e as Map)['Status'] ?? ''}';
        return s == 'open' || s == 'scanning';
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

  Future<void> _startScan() async {
    final task = selected;
    if (task == null) return;
    final id = task['ID'];
    await openContinuousScan(
      context,
      title: '盘点连扫 #$id',
      progressText: '已扫 $scanned',
      onCode: (code) async {
        await runSubmit(
          path: '/inventory/tasks/$id/scan',
          body: {'uniqueCode': code},
          label: '盘点扫码',
        );
        if (!err) {
          scanned++;
          if (mounted) setState(() {});
        } else {
          throw Exception(msg);
        }
      },
    );
  }

  Future<void> _submitTask() async {
    final id = selected?['ID'];
    if (id == null) return;
    setState(() {
      busy = true;
      msg = '';
      err = false;
    });
    try {
      await Api.post('/inventory/tasks/$id/submit', {});
      setState(() => msg = '盘点已提交，等待过账');
      _load();
      setState(() => selected = null);
    } catch (e) {
      setState(() {
        err = true;
        msg = e.toString();
      });
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('盘点扫码'), actions: [IconButton(icon: const Icon(Icons.refresh), onPressed: _load)]),
      body: loading
          ? const Center(child: CircularProgressIndicator())
          : selected == null
              ? ListView(
                  children: [
                    const Padding(
                      padding: EdgeInsets.all(16),
                      child: Text('选择盘点任务', style: TextStyle(fontWeight: FontWeight.w600)),
                    ),
                    TaskPickerList(
                      items: tasks,
                      titleBuilder: (r) => '${r['Name'] ?? '任务'} (#${r['ID']})',
                      subtitleBuilder: (r) => '状态 ${r['Status']} · 仓 ${r['WarehouseID']}',
                      onSelect: (r) => setState(() {
                        selected = r;
                        scanned = 0;
                      }),
                    ),
                    ExecResultBanner(msg: msg, isError: err),
                  ],
                )
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    ListTile(
                      contentPadding: EdgeInsets.zero,
                      title: Text('${selected!['Name']}'),
                      subtitle: Text('状态 ${selected!['Status']}'),
                      trailing: TextButton(onPressed: () => setState(() => selected = null), child: const Text('重选')),
                    ),
                    Text('已扫 $scanned', style: const TextStyle(fontWeight: FontWeight.w600)),
                    const SizedBox(height: 12),
                    FilledButton.icon(
                      onPressed: busy ? null : _startScan,
                      icon: const Icon(Icons.qr_code_scanner),
                      label: const Text('开始连扫'),
                    ),
                    const SizedBox(height: 8),
                    OutlinedButton(onPressed: busy ? null : _submitTask, child: const Text('提交盘点任务')),
                    ExecResultBanner(msg: msg, isError: err),
                  ],
                ),
    );
  }
}
