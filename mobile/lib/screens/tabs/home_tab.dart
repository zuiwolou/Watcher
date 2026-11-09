import 'package:flutter/material.dart';

import '../../services/api.dart';
import '../../services/outbox.dart';
import '../../services/session.dart';
import '../execute/execute_hub.dart';

class HomeTab extends StatefulWidget {
  const HomeTab({super.key, this.onOpenAlarms, this.onOpenExecute});
  final VoidCallback? onOpenAlarms;
  final VoidCallback? onOpenExecute;

  @override
  State<HomeTab> createState() => _HomeTabState();
}

class _HomeTabState extends State<HomeTab> {
  int raised = 0;
  int pendingOutbox = 0;
  List recent = [];
  String userName = '';

  @override
  void initState() {
    super.initState();
    reload();
  }

  Future<void> reload() async {
    final u = await Session.user();
    userName = '${u['displayName'] ?? u['username'] ?? ''}';
    try {
      final alarms = await Api.get('/alarms?status=raised&page=1&pageSize=3');
      raised = (alarms is Map) ? (alarms['total'] as num?)?.toInt() ?? 0 : 0;
      recent = (alarms is Map) ? (alarms['list'] as List? ?? []) : [];
    } catch (_) {}
    final box = await OutboxService.instance.list();
    pendingOutbox = box.where((e) => e.status == OutboxStatus.pending || e.status == OutboxStatus.failed).length;
    if (mounted) setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    return RefreshIndicator(
      onRefresh: reload,
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Text('你好，${userName.isEmpty ? '现场员' : userName}', style: const TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
          const SizedBox(height: 12),
          Row(children: [
            Expanded(child: _MetricCard(title: '未确认告警', value: '$raised', onTap: widget.onOpenAlarms)),
            const SizedBox(width: 8),
            Expanded(child: _MetricCard(title: '待同步', value: '$pendingOutbox', onTap: null)),
          ]),
          const SizedBox(height: 16),
          ListTile(
            contentPadding: EdgeInsets.zero,
            title: const Text('快捷执行', style: TextStyle(fontWeight: FontWeight.w600)),
            trailing: TextButton(onPressed: widget.onOpenExecute, child: const Text('全部')),
          ),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              ActionChip(label: const Text('打开执行中心'), onPressed: widget.onOpenExecute ?? () {}),
              ActionChip(
                label: const Text('执行页'),
                onPressed: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => const ExecuteHub())),
              ),
            ],
          ),
          const SizedBox(height: 16),
          const Text('最新告警', style: TextStyle(fontWeight: FontWeight.w600)),
          const SizedBox(height: 8),
          if (recent.isEmpty) const Text('暂无未确认告警', style: TextStyle(color: Colors.black54)),
          ...recent.map((a) {
            final m = a as Map;
            return Card(
              child: ListTile(
                title: Text('${m['Title']}'),
                subtitle: Text('${m['Level']} · ${m['Status']}'),
                onTap: widget.onOpenAlarms,
              ),
            );
          }),
        ],
      ),
    );
  }
}

class _MetricCard extends StatelessWidget {
  const _MetricCard({required this.title, required this.value, this.onTap});
  final String title;
  final String value;
  final VoidCallback? onTap;
  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: const Color(0xFF102A43),
          borderRadius: BorderRadius.circular(12),
        ),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(title, style: const TextStyle(color: Colors.white70, fontSize: 13)),
          const SizedBox(height: 8),
          Text(value, style: const TextStyle(color: Colors.white, fontSize: 28, fontWeight: FontWeight.bold)),
        ]),
      ),
    );
  }
}
