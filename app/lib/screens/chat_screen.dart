import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../models/message.dart';
import '../services/api_service.dart';
import '../services/discovery_service.dart';
import '../services/voice_service.dart';
import '../widgets/chat_bubble.dart';
import '../widgets/voice_button.dart';

class ChatScreen extends StatefulWidget {
  const ChatScreen({super.key});

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final TextEditingController _textController = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  final List<Message> _messages = [];
  bool _isLoading = false;

  @override
  void initState() {
    super.initState();
    _initialize();
  }

  Future<void> _initialize() async {
    final voiceService = context.read<VoiceService>();

    // Initialize voice
    await voiceService.initialize();

    // Start server discovery
    final discoveryService = context.read<DiscoveryService>();
    discoveryService.startDiscovery();
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollController.hasClients) {
        _scrollController.animateTo(
          _scrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 300),
          curve: Curves.easeOut,
        );
      }
    });
  }

  Future<void> _sendMessage(String text) async {
    if (text.trim().isEmpty) return;

    final apiService = context.read<ApiService>();

    setState(() {
      _messages.add(Message.user(text.trim()));
      _isLoading = true;
    });
    _textController.clear();
    _scrollToBottom();

    String assistantResponse = '';

    await for (final event in apiService.chat(text.trim())) {
      setState(() {
        switch (event.type) {
          case 'text':
            assistantResponse += event.content ?? '';
            // Update or add assistant message
            if (_messages.isNotEmpty && _messages.last.isAssistant) {
              _messages.removeLast();
            }
            _messages.add(Message.assistant(assistantResponse));
            break;
          case 'tool_call':
            _messages.add(Message.toolCall(
              event.name ?? 'unknown',
              event.args,
            ));
            break;
          case 'tool_result':
            _messages.add(Message.toolResult(
              event.name ?? 'unknown',
              event.result,
            ));
            break;
          case 'error':
            _messages.add(Message(
              id: DateTime.now().millisecondsSinceEpoch.toString(),
              role: 'error',
              content: event.content ?? 'Unknown error',
              timestamp: DateTime.now(),
            ));
            break;
          case 'done':
            _isLoading = false;
            break;
        }
      });
      _scrollToBottom();
    }

    setState(() => _isLoading = false);
  }

  void _showSettingsSheet() {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (context) => const _SettingsSheet(),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Pocket Assistant'),
        actions: [
          Consumer<ApiService>(
            builder: (context, api, _) => Icon(
              Icons.circle,
              size: 12,
              color: api.isConnected ? Colors.green : Colors.red,
            ),
          ),
          const SizedBox(width: 8),
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () {
              final api = context.read<ApiService>();
              api.newConversation();
              setState(() => _messages.clear());
            },
            tooltip: 'New conversation',
          ),
          IconButton(
            icon: const Icon(Icons.settings),
            onPressed: _showSettingsSheet,
            tooltip: 'Settings',
          ),
        ],
      ),
      body: Column(
        children: [
          // Connection status banner
          Consumer<ApiService>(
            builder: (context, api, _) {
              if (api.isConnected) return const SizedBox.shrink();
              return Container(
                color: Colors.orange.shade900,
                padding: const EdgeInsets.all(8),
                child: Row(
                  children: [
                    const Icon(Icons.warning, size: 16),
                    const SizedBox(width: 8),
                    const Expanded(
                      child: Text('Not connected to server'),
                    ),
                    TextButton(
                      onPressed: _showSettingsSheet,
                      child: const Text('Connect'),
                    ),
                  ],
                ),
              );
            },
          ),

          // Chat messages
          Expanded(
            child: _messages.isEmpty
                ? const Center(
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Icon(Icons.assistant, size: 64, color: Colors.grey),
                        SizedBox(height: 16),
                        Text(
                          'Say something or type a message',
                          style: TextStyle(color: Colors.grey),
                        ),
                      ],
                    ),
                  )
                : ListView.builder(
                    controller: _scrollController,
                    padding: const EdgeInsets.all(16),
                    itemCount: _messages.length,
                    itemBuilder: (context, index) {
                      return ChatBubble(message: _messages[index]);
                    },
                  ),
          ),

          // Loading indicator
          if (_isLoading)
            const Padding(
              padding: EdgeInsets.symmetric(horizontal: 16),
              child: LinearProgressIndicator(),
            ),

          // Input area
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.surface,
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withOpacity(0.1),
                  blurRadius: 4,
                  offset: const Offset(0, -2),
                ),
              ],
            ),
            child: SafeArea(
              child: Row(
                children: [
                  // Voice button
                  VoiceButton(
                    onResult: _sendMessage,
                  ),
                  const SizedBox(width: 8),

                  // Text input
                  Expanded(
                    child: TextField(
                      controller: _textController,
                      decoration: const InputDecoration(
                        hintText: 'Type a message...',
                        border: OutlineInputBorder(),
                        contentPadding: EdgeInsets.symmetric(
                          horizontal: 16,
                          vertical: 12,
                        ),
                      ),
                      textInputAction: TextInputAction.send,
                      onSubmitted: _sendMessage,
                    ),
                  ),
                  const SizedBox(width: 8),

                  // Send button
                  IconButton.filled(
                    onPressed: () => _sendMessage(_textController.text),
                    icon: const Icon(Icons.send),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  @override
  void dispose() {
    _textController.dispose();
    _scrollController.dispose();
    super.dispose();
  }
}

class _SettingsSheet extends StatelessWidget {
  const _SettingsSheet();

  @override
  Widget build(BuildContext context) {
    return DraggableScrollableSheet(
      initialChildSize: 0.7,
      minChildSize: 0.5,
      maxChildSize: 0.95,
      expand: false,
      builder: (context, scrollController) {
        return SingleChildScrollView(
          controller: scrollController,
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Handle
              Center(
                child: Container(
                  width: 40,
                  height: 4,
                  decoration: BoxDecoration(
                    color: Colors.grey,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
              const SizedBox(height: 16),

              // Title
              Text(
                'Settings',
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              const SizedBox(height: 24),

              // Server section
              Text(
                'Server Connection',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 8),
              _ServerSection(),
            ],
          ),
        );
      },
    );
  }
}

class _ServerSection extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Consumer2<DiscoveryService, ApiService>(
      builder: (context, discovery, api, _) {
        return Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Status
                Row(
                  children: [
                    Icon(
                      api.isConnected ? Icons.check_circle : Icons.error,
                      color: api.isConnected ? Colors.green : Colors.red,
                    ),
                    const SizedBox(width: 8),
                    Text(api.isConnected ? 'Connected' : 'Disconnected'),
                  ],
                ),
                const SizedBox(height: 16),

                // Discovered servers
                if (discovery.servers.isNotEmpty) ...[
                  const Text('Discovered servers:'),
                  const SizedBox(height: 8),
                  ...discovery.servers.map((server) => ListTile(
                        title: Text(server.name),
                        subtitle: Text('${server.host}:${server.port}'),
                        trailing: discovery.selectedServer == server
                            ? const Icon(Icons.check)
                            : null,
                        onTap: () {
                          discovery.selectServer(server);
                          api.setServer(server.url);
                          api.checkHealth();
                        },
                      )),
                ] else if (discovery.isSearching) ...[
                  const Row(
                    children: [
                      SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      ),
                      SizedBox(width: 8),
                      Text('Searching for servers...'),
                    ],
                  ),
                ] else ...[
                  const Text('No servers found'),
                ],

                const SizedBox(height: 16),

                // Manual connection
                Row(
                  children: [
                    Expanded(
                      child: OutlinedButton.icon(
                        onPressed: () => discovery.startDiscovery(),
                        icon: const Icon(Icons.refresh),
                        label: const Text('Refresh'),
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: OutlinedButton.icon(
                        onPressed: () => _showManualDialog(context),
                        icon: const Icon(Icons.edit),
                        label: const Text('Manual'),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        );
      },
    );
  }

  void _showManualDialog(BuildContext context) {
    final hostController = TextEditingController();
    final portController = TextEditingController(text: '8443');

    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Manual Connection'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: hostController,
              decoration: const InputDecoration(
                labelText: 'Host',
                hintText: '192.168.1.100',
              ),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: portController,
              decoration: const InputDecoration(
                labelText: 'Port',
              ),
              keyboardType: TextInputType.number,
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              final host = hostController.text.trim();
              final port = int.tryParse(portController.text) ?? 8443;
              if (host.isNotEmpty) {
                final discovery = context.read<DiscoveryService>();
                final api = context.read<ApiService>();
                discovery.setManualServer(host, port);
                api.setServer('https://$host:$port');
                api.checkHealth();
                Navigator.pop(context);
              }
            },
            child: const Text('Connect'),
          ),
        ],
      ),
    );
  }
}

