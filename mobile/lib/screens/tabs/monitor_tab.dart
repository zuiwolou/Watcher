import 'package:flutter/material.dart';

import '../../services/api.dart';

class MonitorTab extends StatefulWidget {
  const MonitorTab({super.key});
  @override
  State<MonitorTab> createState() => _MonitorTabState();
}

class _MonitorTabState extends State<MonitorTab> {
  String warehouseId = 'WH01';
  String metric = 'temperature';
  List rows = [];

  Future<void> load() async {
    final data = await Api.get('/history/telemetry?warehouseId=$warehouseId&metric=$metric');
    setState(() => rows = data is List ? data : []);
  }

  @override
  void initState() {
    super.initState();
    load();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            children: [
              Expanded(
                child: TextField(
                  decoration: const InputDecoration(labelText: '仓ID', isDense: true),
                  controller: TextEditingController(text: warehouseId),
                  onChanged: (v) => warehouseId = v,
                ),
              ),
              const SizedBox(width: 8),
              DropdownButton<String>(
                value: metric,
                items: const [
                  DropdownMenuItem(value: 'temperature', child: Text('温度')),
                  DropdownMenuItem(value: 'humidity', child: Text('湿度')),
                ],
                onChanged: (v) => setState(() => metric = v ?? 'temperature'),
              ),
              IconButton(onPressed: load, icon: const Icon(Icons.refresh)),
            ],
          ),
        ),
        Expanded(
          child: ListView.builder(
            itemCount: rows.length,
            itemBuilder: (_, i) {
              final r = rows[i] as Map;
              return ListTile(
                title: Text('${r['Metric'] ?? metric}: ${r['Value']}'),
                subtitle: Text('${r['DeviceID'] ?? ''} · ${r['Ts'] ?? ''}'),
              );
            },
          ),
        ),
      ],
    );
  }
}
