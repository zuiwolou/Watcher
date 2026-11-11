import 'package:flutter/material.dart';

import '../../services/api.dart';
import 'exec_forms.dart';

class InboundPage extends StatefulWidget {
  const InboundPage({super.key});
  @override
  State<InboundPage> createState() => _InboundPageState();
}

class _InboundPageState extends State<InboundPage> with ExecSubmitMixin {
  List orders = [];
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
      final data = await Api.get('/inbound/orders?page=1&pageSize=50');
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
    await openContinuousScan(
      context,
      title: '入库连扫 #$id',
      progressText: '已扫 $scanned · 单据 $id',
      onCode: (code) async {
        await runSubmit(
          path: '/inbound/orders/$id/confirm',
          body: {'uniqueCode': code},
          label: '入库确认',
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
      appBar: AppBar(title: const Text('入库确认'), actions: [IconButton(icon: const Icon(Icons.refresh), onPressed: _load)]),
      body: loading
          ? const Center(child: CircularProgressIndicator())
          : selected == null
              ? ListView(
                  children: [
                    const Padding(
                      padding: EdgeInsets.all(16),
                      child: Text('选择待确认入库单', style: TextStyle(fontWeight: FontWeight.w600)),
                    ),
                    TaskPickerList(
                      items: orders,
                      titleBuilder: (r) => '单号 ${r['OrderNo'] ?? r['ID']}',
                      subtitleBuilder: (r) => '状态 ${r['Status']} · 仓 ${r['WarehouseID']}',
                      onSelect: (r) => setState(() {
                        selected = r;
                        scanned = 0;
                      }),
                      emptyText: '暂无待确认入库单',
                    ),
                    ExecResultBanner(msg: msg, isError: err),
                  ],
                )
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    ListTile(
                      contentPadding: EdgeInsets.zero,
                      title: Text('入库单 ${selected!['OrderNo'] ?? selected!['ID']}'),
                      subtitle: Text('状态 ${selected!['Status']}'),
                      trailing: TextButton(onPressed: () => setState(() => selected = null), child: const Text('重选')),
                    ),
                    Text('已扫 $scanned', style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                    LocationPhotoBar(fix: fix, photoPath: photoPath, onLocate: () => refreshLocation(), onPhoto: capturePhoto),
                    const SizedBox(height: 12),
                    FilledButton.icon(
                      onPressed: busy ? null : _startScan,
                      icon: const Icon(Icons.qr_code_scanner),
                      label: const Text('开始连扫'),
                    ),
                    ExecResultBanner(msg: msg, isError: err),
                  ],
                ),
    );
  }
}
