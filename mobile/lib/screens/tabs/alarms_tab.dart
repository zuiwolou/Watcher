import 'package:flutter/material.dart';

import '../../services/api.dart';
import '../../services/session.dart';

class AlarmsTab extends StatefulWidget {
  const AlarmsTab({super.key});
  @override
  State<AlarmsTab> createState() => _AlarmsTabState();
}

class _AlarmsTabState extends State<AlarmsTab> {
  List list = [];
  bool canAck = false;

  @override
  void initState() {
    super.initState();
    Session.has('alarm:ack').then((v) => setState(() => canAck = v));
    reload();
  }

  Future<void> reload() async {
    final data = await Api.get('/alarms?page=1&pageSize=50');
    setState(() => list = (data['list'] as List?) ?? []);
  }

  @override
  Widget build(BuildContext context) {
    return RefreshIndicator(
      onRefresh: reload,
      child: ListView.builder(
        itemCount: list.length,
        itemBuilder: (_, i) {
          final a = list[i] as Map;
          final raised = a['Status'] == 'raised';
          return ListTile(
            leading: Container(
              width: 6,
              color: raised ? Colors.red : Colors.grey,
            ),
            title: Text('${a['Title']}'),
            subtitle: Text('${a['Status']} · ${a['Level']} · ${a['DeviceID'] ?? ''}'),
            trailing: raised && canAck
                ? TextButton(
                    child: const Text('确认'),
                    onPressed: () async {
                      await Api.post('/alarms/${a['ID']}/ack', {});
                      reload();
                    },
                  )
                : null,
          );
        },
      ),
    );
  }
}
