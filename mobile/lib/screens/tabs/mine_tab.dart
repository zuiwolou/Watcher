import 'package:flutter/material.dart';
import 'package:package_info_plus/package_info_plus.dart';

import '../../services/api.dart';
import '../../services/outbox.dart';
import '../../services/push_service.dart';
import '../../services/realtime_service.dart';
import '../../services/session.dart';
import '../change_password_page.dart';
import '../login_page.dart';
import '../readonly/geofence_page.dart';
import '../readonly/materials_page.dart';

class MineTab extends StatefulWidget {
  const MineTab({super.key});
  @override
  State<MineTab> createState() => _MineTabState();
}

class _MineTabState extends State<MineTab> {
  Map user = {};
  List<String> perms = [];
  List<OutboxItem> box = [];
  String? pushToken;
  String version = '';

  @override
  void initState() {
    super.initState();
    _load();
    OutboxService.instance.changes.listen((items) {
      if (mounted) setState(() => box = items);
    });
  }

  Future<void> _load() async {
    user = await Session.user();
    perms = await Session.permissions();
    box = await OutboxService.instance.list();
    pushToken = PushService.instance.deviceToken;
    try {
      final info = await PackageInfo.fromPlatform();
      version = '${info.version}+${info.buildNumber}';
    } catch (_) {
      version = '0.1.0';
    }
    setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        ListTile(
          leading: const CircleAvatar(child: Icon(Icons.person)),
          title: Text('${user['displayName'] ?? user['username'] ?? '-'}'),
          subtitle: Text('账号 ${user['username'] ?? '-'}'),
        ),
        const Divider(),
        const Text('权限摘要', style: TextStyle(fontWeight: FontWeight.w600)),
        Wrap(
          spacing: 6,
          children: perms.take(24).map((p) => Chip(label: Text(p, style: const TextStyle(fontSize: 11)))).toList(),
        ),
        const SizedBox(height: 8),
        if (perms.contains('material:view'))
          ListTile(
            leading: const Icon(Icons.inventory),
            title: const Text('物资/唯一码查询'),
            onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => const MaterialsPage())),
          ),
        if (perms.contains('geofence:view') || perms.contains('track:view'))
          ListTile(
            leading: const Icon(Icons.map_outlined),
            title: const Text('围栏/地图（只读）'),
            onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => const GeofencePage())),
          ),
        ListTile(
          leading: const Icon(Icons.lock_outline),
          title: const Text('修改密码'),
          onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => const ChangePasswordPage())),
        ),
        ListTile(
          leading: const Icon(Icons.cleaning_services_outlined),
          title: const Text('清理展示缓存'),
          onTap: () async {
            await Session.clearCacheMeta();
            if (!context.mounted) return;
            ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('已清理缓存时间戳')));
          },
        ),
        ListTile(
          leading: const Icon(Icons.info_outline),
          title: const Text('关于'),
          subtitle: Text('Watcher APP $version'),
        ),
        ListTile(
          title: const Text('推送设备 Token'),
          subtitle: Text(pushToken ?? '-', maxLines: 2, overflow: TextOverflow.ellipsis),
          trailing: IconButton(
            icon: const Icon(Icons.notifications_active),
            onPressed: () => PushService.instance.showSummary(title: '测试通知', body: '本地推送通道正常'),
          ),
        ),
        const Divider(),
        Row(
          children: [
            const Text('离线 Outbox', style: TextStyle(fontWeight: FontWeight.w600)),
            const Spacer(),
            TextButton(onPressed: () => OutboxService.instance.flush(), child: const Text('立即同步')),
          ],
        ),
        if (box.isEmpty) const Text('队列为空', style: TextStyle(color: Colors.black54)),
        ...box.map((item) => Card(
              child: ListTile(
                title: Text(item.label.isEmpty ? item.path : item.label),
                subtitle: Text('${item.status.name} · 尝试${item.attempts}\n${item.lastError}', maxLines: 3),
                isThreeLine: true,
                trailing: Wrap(
                  children: [
                    if (item.status == OutboxStatus.failed || item.status == OutboxStatus.pending)
                      IconButton(icon: const Icon(Icons.refresh), onPressed: () => OutboxService.instance.retry(item.id)),
                    IconButton(icon: const Icon(Icons.delete_outline), onPressed: () => OutboxService.instance.remove(item.id)),
                  ],
                ),
              ),
            )),
        const SizedBox(height: 24),
        FilledButton(
          onPressed: () async {
            OutboxService.instance.stop();
            PushService.instance.stop();
            await RealtimeService.instance.stop();
            final rt = await Session.refreshToken();
            try {
              await Api.post('/auth/logout', {'refreshToken': rt ?? ''});
            } catch (_) {}
            await Session.clear();
            if (!context.mounted) return;
            Navigator.of(context).pushAndRemoveUntil(
              MaterialPageRoute(builder: (_) => const LoginPage()),
              (_) => false,
            );
          },
          child: const Text('退出登录'),
        ),
      ],
    );
  }
}
