import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'services/api_service.dart';
import 'services/discovery_service.dart';
import 'services/voice_service.dart';
import 'screens/chat_screen.dart';

void main() {
  runApp(const PocketAssistantApp());
}

class PocketAssistantApp extends StatelessWidget {
  const PocketAssistantApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider(create: (_) => DiscoveryService()),
        ChangeNotifierProvider(create: (_) => ApiService()),
        ChangeNotifierProvider(create: (_) => VoiceService()),
      ],
      child: MaterialApp(
        title: 'Pocket Assistant',
        theme: ThemeData(
          colorScheme: ColorScheme.fromSeed(
            seedColor: Colors.blue,
            brightness: Brightness.dark,
          ),
          useMaterial3: true,
        ),
        home: const ChatScreen(),
        debugShowCheckedModeBanner: false,
      ),
    );
  }
}
