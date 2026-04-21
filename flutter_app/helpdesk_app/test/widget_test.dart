import 'package:flutter_test/flutter_test.dart';
import 'package:helpdesk_app/main.dart';

void main() {
  testWidgets('shows converted helpdesk login screen', (tester) async {
    await tester.pumpWidget(const HelpdeskApp());

    expect(find.text('Helpdesk\nTicketing'), findsOneWidget);
    expect(find.text('Masuk'), findsOneWidget);
    expect(find.text('Buat akun baru'), findsOneWidget);
  });
}
