import 'package:flutter/material.dart';

import '../../services/api.dart';
import 'exec_forms.dart';

class InspectionPage extends StatefulWidget {
  const InspectionPage({super.key});
  @override
  State<InspectionPage> createState() => _InspectionPageState();
}

class _InspectionPageState extends State<InspectionPage> with ExecSubmitMixin {
  List tasks = [];
  List points = [];
  Map? selected;
  Map? selectedPoint;
  String pointStatus = 'done';
  final remark = TextEditingController();
  bool loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => loading = true);
    try {
      final data = await Api.get('/inspection/tasks?page=1&pageSize=50');
      final list = (data is Map) ? (data['list'] as List? ?? []) : <dynamic>[];
      tasks = list.where((e) {
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

  Future<void> _selectTask(Map row) async {
    selected = row;
    selectedPoint = null;
    points = [];
    setState(() {});
    try {
      final routeId = row['RouteID'];
      if (routeId != null) {
        final list = await Api.get('/inspection/routes/$routeId/points');
        points = list is List ? list : [];
        points.sort((a, b) {
          final sa = (a as Map)['Seq'] as int? ?? 0;
          final sb = (b as Map)['Seq'] as int? ?? 0;
          return sa.compareTo(sb);
        });
        // 默认选中序最小的点（后端 ForceOrder 时必须按序）
        if (points.isNotEmpty) {
          selectedPoint = Map<String, dynamic>.from(points.first as Map);
        }
        setState(() {});
      }
    } catch (e) {
      setState(() {
        err = true;
        msg = e.toString();
      });
    }
  }

  Future<void> _submit() async {
    final taskId = selected?['ID'];
    final pointId = selectedPoint?['ID'];
    if (taskId == null || pointId == null) return;
    try {
      await runSubmit(
        path: '/inspection/tasks/$taskId/checkin',
        body: {
          'pointId': pointId,
          'status': pointStatus,
          if (remark.text.trim().isNotEmpty) 'remark': remark.text.trim(),
        },
        label: '巡检打卡',
        requireLocation: true,
        requirePhoto: selectedPoint?['RequirePhoto'] == true,
      );
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('巡检打卡'), actions: [IconButton(icon: const Icon(Icons.refresh), onPressed: _load)]),
      body: loading
          ? const Center(child: CircularProgressIndicator())
          : selected == null
              ? ListView(
                  children: [
                    const Padding(
                      padding: EdgeInsets.all(16),
                      child: Text('选择巡检任务', style: TextStyle(fontWeight: FontWeight.w600)),
                    ),
                    TaskPickerList(
                      items: tasks,
                      titleBuilder: (r) => '任务 #${r['ID']}',
                      subtitleBuilder: (r) => '状态 ${r['Status']} · 路线 ${r['RouteID']}',
                      onSelect: _selectTask,
                    ),
                    ExecResultBanner(msg: msg, isError: err),
                  ],
                )
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    ListTile(
                      contentPadding: EdgeInsets.zero,
                      title: Text('任务 #${selected!['ID']}'),
                      trailing: TextButton(onPressed: () => setState(() => selected = null), child: const Text('重选')),
                    ),
                    const Text('巡检点（按序打卡）', style: TextStyle(fontWeight: FontWeight.w600)),
                    const Text('后端默认强制顺序；跳点会被拒绝', style: TextStyle(fontSize: 12, color: Colors.black54)),
                    ...points.map((p) {
                      final m = Map<String, dynamic>.from(p as Map);
                      final sel = selectedPoint?['ID'] == m['ID'];
                      return ListTile(
                        selected: sel,
                        title: Text('${m['Name'] ?? '点'} · 序 ${m['Seq'] ?? '-'}'),
                        subtitle: Text(m['RequirePhoto'] == true ? '必拍' : '可选拍照'),
                        onTap: () => setState(() => selectedPoint = m),
                      );
                    }),
                    if (selectedPoint != null) ...[
                      DropdownButtonFormField<String>(
                        initialValue: pointStatus,
                        items: const [
                          DropdownMenuItem(value: 'done', child: Text('正常')),
                          DropdownMenuItem(value: 'abnormal', child: Text('异常')),
                        ],
                        onChanged: (v) => setState(() => pointStatus = v ?? 'done'),
                        decoration: const InputDecoration(labelText: '结果'),
                      ),
                      TextField(controller: remark, decoration: const InputDecoration(labelText: '说明')),
                      LocationPhotoBar(
                        fix: fix,
                        photoPath: photoPath,
                        onLocate: () => refreshLocation(required: true),
                        onPhoto: capturePhoto,
                        requirePhotoHint: selectedPoint?['RequirePhoto'] == true,
                      ),
                      const SizedBox(height: 12),
                      FilledButton(onPressed: busy ? null : _submit, child: const Text('提交打卡')),
                    ],
                    ExecResultBanner(msg: msg, isError: err),
                  ],
                ),
    );
  }
}
