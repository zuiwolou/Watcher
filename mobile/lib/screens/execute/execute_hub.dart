import 'package:flutter/material.dart';

import '../../services/session.dart';
import 'inbound_page.dart';
import 'inspection_page.dart';
import 'inventory_page.dart';
import 'outbound_page.dart';
import 'transport_page.dart';

class ExecuteHub extends StatefulWidget {
  const ExecuteHub({super.key});
  @override
  State<ExecuteHub> createState() => _ExecuteHubState();
}

class _ExecuteHubState extends State<ExecuteHub> {
  List<String> perms = [];

  @override
  void initState() {
    super.initState();
    Session.permissions().then((p) => setState(() => perms = p));
  }

  bool has(String p) => perms.contains(p);

  @override
  Widget build(BuildContext context) {
    final cards = <_ExecCard>[];
    if (has('inbound:confirm')) {
      cards.add(_ExecCard('入库确认', Icons.qr_code_scanner, () => const InboundPage()));
    }
    if (has('outbound:confirm')) {
      cards.add(_ExecCard('出库/签收', Icons.local_shipping, () => const OutboundPage()));
    }
    if (has('checkin:exec')) {
      cards.add(_ExecCard('运输打卡', Icons.pin_drop, () => const TransportPage()));
    }
    if (has('inventory:scan')) {
      cards.add(_ExecCard('盘点扫码', Icons.inventory_2, () => const InventoryPage()));
    }
    if (has('inspection:exec')) {
      cards.add(_ExecCard('巡检打卡', Icons.fact_check, () => const InspectionPage()));
    }

    if (cards.isEmpty) {
      return const Center(child: Text('暂无执行权限'));
    }

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        const Text('现场执行', style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
        const SizedBox(height: 8),
        const Text('扫码 / 拍照 / 定位；弱网自动入 Outbox', style: TextStyle(color: Colors.black54)),
        const SizedBox(height: 16),
        ...cards.map((c) => Card(
              child: ListTile(
                leading: Icon(c.icon, size: 32, color: const Color(0xFF0B3D4A)),
                title: Text(c.title, style: const TextStyle(fontWeight: FontWeight.w600)),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => c.builder())),
              ),
            )),
      ],
    );
  }
}

class _ExecCard {
  _ExecCard(this.title, this.icon, this.builder);
  final String title;
  final IconData icon;
  final Widget Function() builder;
}
