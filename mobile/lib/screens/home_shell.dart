import 'package:flutter/material.dart';

import '../services/outbox.dart';
import '../services/push_service.dart';
import '../services/session.dart';
import 'execute/execute_hub.dart';
import 'tabs/alarms_tab.dart';
import 'tabs/home_tab.dart';
import 'tabs/mine_tab.dart';
import 'tabs/monitor_tab.dart';

class HomeShell extends StatefulWidget {
  const HomeShell({super.key});
  @override
  State<HomeShell> createState() => _HomeShellState();
}

class _HomeShellState extends State<HomeShell> {
  int idx = 0;
  List<String> perms = [];

  @override
  void initState() {
    super.initState();
    Session.permissions().then((p) => setState(() => perms = p));
    OutboxService.instance.startAutoFlush();
    PushService.instance.startAlarmPolling();
  }

  bool has(String p) => perms.contains(p);

  bool get canExecute =>
      has('inbound:confirm') ||
      has('outbound:confirm') ||
      has('checkin:exec') ||
      has('inventory:scan') ||
      has('inspection:exec');

  @override
  Widget build(BuildContext context) {
    final tabs = <Widget>[];
    final items = <BottomNavigationBarItem>[];

    // 首页
    tabs.add(HomeTab(
      onOpenAlarms: () {
        final i = items.indexWhere((e) => e.label == '告警');
        if (i >= 0) setState(() => idx = i);
      },
      onOpenExecute: () {
        final i = items.indexWhere((e) => e.label == '执行');
        if (i >= 0) {
          setState(() => idx = i);
        } else {
          Navigator.of(context).push(MaterialPageRoute(builder: (_) => const ExecuteHub()));
        }
      },
    ));
    items.add(const BottomNavigationBarItem(icon: Icon(Icons.home_outlined), label: '首页'));

    if (has('alarm:view')) {
      tabs.add(const AlarmsTab());
      items.add(const BottomNavigationBarItem(icon: Icon(Icons.warning_amber), label: '告警'));
    }
    if (has('monitor:view')) {
      tabs.add(const MonitorTab());
      items.add(const BottomNavigationBarItem(icon: Icon(Icons.show_chart), label: '监测'));
    }
    if (canExecute) {
      tabs.add(const ExecuteHub());
      items.add(const BottomNavigationBarItem(icon: Icon(Icons.play_circle_outline), label: '执行'));
    }
    tabs.add(const MineTab());
    items.add(const BottomNavigationBarItem(icon: Icon(Icons.person_outline), label: '我的'));

    final safeIdx = idx.clamp(0, tabs.length - 1);
    return Scaffold(
      appBar: AppBar(title: const Text('危废管控')),
      body: IndexedStack(index: safeIdx, children: tabs),
      bottomNavigationBar: BottomNavigationBar(
        type: BottomNavigationBarType.fixed,
        currentIndex: safeIdx,
        onTap: (i) => setState(() => idx = i),
        items: items,
      ),
    );
  }
}
