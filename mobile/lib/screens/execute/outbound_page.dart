import 'package:flutter/material.dart';

import '../../services/api.dart';
import 'exec_forms.dart';

class OutboundPage extends StatefulWidget {
  const OutboundPage({super.key});
  @override
  State<OutboundPage> createState() => _OutboundPageState();
}

class _OutboundPageState extends State<OutboundPage> with ExecSubmitMixin {
  List orders = [];
  Map? selected;
  String action = 'lock';
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
      final data = await Api.get('/outbound/orders?page=1&pageSize=50');
      final list = (data is Map) ? (data['list'] as List? ?? []) : <dynamic>[];
      orders = list.where((e) {
        final s = '${(e as Map)['Status'] ?? ''}';
        return s == 'open' || s == 'in_progress';
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
    final order = selected;
    if (order == null) return;
    final id = order['ID'];
    final needLoc = action == 'receive';
    await openContinuousScan(
      context,
      title: action == 'lock' ? '出库锁定连扫' : '签收连扫',
      progressText: '已扫 $scanned · 动作 $action',
      onCode: (code) async {
        await runSubmit(
          path: '/outbound/orders/$id/confirm',
          body: {'uniqueCode': code, 'action': action},
          label: action == 'lock' ? '出库锁定' : '签收',
          requireLocation: needLoc,
          requirePhoto: needLoc,
        );
        if (!err) {
          scanned++;
          if (mounted) setState(() {});
        } else {
          throw Exception(msg);
        }
      },
    );
    _load();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('出库/签收'), actions: [IconButton(icon: const Icon(Icons.refresh), onPressed: _load)]),
      body: loading
          ? const Center(child: CircularProgressIndicator())
          : selected == null
              ? ListView(
                  children: [
                    const Padding(
                      padding: EdgeInsets.all(16),
                      child: Text('选择出库单', style: TextStyle(fontWeight: FontWeight.w600)),
                    ),
                    TaskPickerList(
                      items: orders,
                      titleBuilder: (r) => '单号 ${r['OrderNo'] ?? r['ID']}',
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
                      title: Text('出库单 ${selected!['OrderNo'] ?? selected!['ID']}'),
                      trailing: TextButton(onPressed: () => setState(() => selected = null), child: const Text('重选')),
                    ),
                    DropdownButtonFormField<String>(
                      initialValue: action,
                      items: const [
                        DropdownMenuItem(value: 'lock', child: Text('锁定')),
                        DropdownMenuItem(value: 'receive', child: Text('签收')),
                      ],
                      onChanged: (v) => setState(() => action = v ?? 'lock'),
                      decoration: const InputDecoration(labelText: '动作'),
                    ),
                    Text('已扫 $scanned', style: const TextStyle(fontWeight: FontWeight.w600)),
                    LocationPhotoBar(
                      fix: fix,
                      photoPath: photoPath,
                      onLocate: () => refreshLocation(required: action == 'receive'),
                      onPhoto: capturePhoto,
                      requirePhotoHint: action == 'receive',
                    ),
                    const SizedBox(height: 12),
                    FilledButton.icon(
                      onPressed: busy ? null : _startScan,
                      icon: const Icon(Icons.qr_code_scanner),
                      label: Text(action == 'lock' ? '开始锁定连扫' : '开始签收连扫'),
                    ),
                    ExecResultBanner(msg: msg, isError: err),
                  ],
                ),
    );
  }
}
