import 'package:flutter_test/flutter_test.dart';
import 'package:watcher_app/main.dart';

void main() {
  testWidgets('app boots to gate', (tester) async {
    await tester.pumpWidget(const WatcherApp());
    expect(find.byType(GatePage), findsOneWidget);
  });
}
