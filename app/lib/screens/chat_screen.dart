import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../models/message.dart';
import '../services/api_service.dart';
import '../services/discovery_service.dart';
import '../services/mode_service.dart';
import '../services/voice_service.dart';
import '../widgets/chat_bubble.dart';
import '../widgets/mode_tabs.dart';
import '../widgets/voice_button.dart';
import '../widgets/llm_settings_section.dart';
import '../widgets/location_settings_section.dart';
import '../widgets/loading_shimmer.dart';

export '../widgets/loading_shimmer.dart';

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
  bool _isAutoScrolling = false;
  Timer? _streamThrottleTimer;
  bool _streamUpdatePending = false;

  @override
  void initState() {
    super.initState();
    _initialize();

    // Listen for API connection changes to fetch modes
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final apiService = context.read<ApiService>();
      apiService.addListener(_onApiServiceChange);
    });
  }

  void _onApiServiceChange() async {
    final apiService = context.read<ApiService>();
    final modeService = context.read<ModeService>();

    if (apiService.isConnected && apiService.baseUrl != null) {
      // Fetch modes if not already loaded
      if (modeService.availableModes.isEmpty && !modeService.isLoading) {
        debugPrint('Fetching modes from ${apiService.baseUrl}');
        try {
          await modeService.fetchModes(apiService.baseUrl!);
        } catch (e) {
          debugPrint('Failed to fetch modes: $e');
        }
      }
      // Set the mode if available and different from current
      if (modeService.selectedModeId != null &&
          apiService.selectedMode != modeService.selectedModeId) {
        apiService.setMode(modeService.selectedModeId);
      }
    }
  }

  Future<void> _initialize() async {
    final voiceService = context.read<VoiceService>();
    final apiService = context.read<ApiService>();
    final modeService = context.read<ModeService>();

    // Initialize voice
    await voiceService.initialize();

    // Fetch modes when connected
    if (apiService.isConnected && apiService.baseUrl != null) {
      await modeService.fetchModes(apiService.baseUrl!);
      // Set initial mode
      if (modeService.selectedModeId != null) {
        apiService.setMode(modeService.selectedModeId);
      }
    }

    // Start server discovery and auto-connect when found
    if (!mounted) return;
    final discoveryService = context.read<DiscoveryService>();
    discoveryService.addListener(() {
      if (!mounted) return;
      if (!apiService.isConnected && discoveryService.selectedServer != null) {
        final server = discoveryService.selectedServer!;
        apiService.setServer(server.url);
        apiService.checkHealth();
      }
    });
    discoveryService.startDiscovery();
  }

  void _scrollToBottom() {
    // Debounce scroll to avoid excessive animations
    if (_isAutoScrolling) return;

    _isAutoScrolling = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollController.hasClients) {
        _scrollController
            .animateTo(
          _scrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        )
            .then((_) {
          _isAutoScrolling = false;
        });
      } else {
        _isAutoScrolling = false;
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
      switch (event.type) {
        case 'text':
          assistantResponse += event.content ?? '';

          // Timer-based throttle: update UI at most every 100ms
          if (!_streamUpdatePending) {
            _streamUpdatePending = true;
            _streamThrottleTimer?.cancel();
            _streamThrottleTimer = Timer(const Duration(milliseconds: 100), () {
              _streamUpdatePending = false;
              if (mounted) {
                setState(() {
                  if (_messages.isNotEmpty && _messages.last.isAssistant) {
                    _messages.removeLast();
                  }
                  _messages.add(Message.assistant(assistantResponse));
                });
                _scrollToBottom();
              }
            });
          }
          break;
        case 'thinking':
          setState(() {
            _messages.add(Message.thinking(event.content ?? ''));
          });
          _scrollToBottom();
          break;
        case 'tool_call':
          setState(() {
            _messages.add(Message.toolCall(
              event.name ?? 'unknown',
              event.args,
            ));
          });
          _scrollToBottom();
          break;
        case 'tool_result':
          setState(() {
            _messages.add(Message.toolResult(
              event.name ?? 'unknown',
              event.result,
            ));
          });
          _scrollToBottom();
          break;
        case 'error':
          setState(() {
            _messages.add(Message(
              id: DateTime.now().millisecondsSinceEpoch.toString(),
              role: 'error',
              content: event.content ?? 'Unknown error',
              timestamp: DateTime.now(),
            ));
          });
          _scrollToBottom();
          break;
        case 'done':
          // Cancel any pending throttled update and do final flush
          _streamThrottleTimer?.cancel();
          _streamUpdatePending = false;
          setState(() {
            if (assistantResponse.isNotEmpty) {
              // Remove previous partial assistant message if exists
              if (_messages.isNotEmpty && _messages.last.isAssistant) {
                _messages.removeLast();
              }
              _messages.add(Message.assistant(assistantResponse));
            }
            _isLoading = false;
          });
          _scrollToBottom();
          break;
      }
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
        title: const Text('Nexus'),
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
          // Mode selection tabs
          const ModeTabs(),

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
                    // Add cache extent for smoother scrolling
                    cacheExtent: 500,
                    itemBuilder: (context, index) {
                      final message = _messages[index];
                      // Use RepaintBoundary to isolate rebuilds for each bubble
                      return RepaintBoundary(
                        child: ChatBubble(
                          key: ValueKey(message.id),
                          message: message,
                        ),
                      );
                    },
                  ),
          ),

          // Loading indicator - show typing indicator
          if (_isLoading)
            const Padding(
              padding: EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              child: TypingIndicator(),
            ),

          // Input area
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.surface,
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withValues(alpha: 0.1),
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
    _streamThrottleTimer?.cancel();
    final apiService = context.read<ApiService>();
    apiService.removeListener(_onApiServiceChange);
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

              const SizedBox(height: 24),

              // LLM Provider section
              Text(
                'Language Model',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 8),
              const LLMSettingsSection(),

              const SizedBox(height: 24),

              // Location section
              const LocationSettingsSection(),
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
                        dense: true,
                        title: Text(server.name),
                        subtitle: Text('${server.host}:${server.port}'),
                        trailing: discovery.selectedServer == server
                            ? const Icon(Icons.check, color: Colors.green)
                            : null,
                        onTap: () async {
                          discovery.selectServer(server);
                          api.setServer(server.url);
                          // Show loading snackbar while connecting
                          ScaffoldMessenger.of(context).showSnackBar(
                            const SnackBar(
                              content: Row(
                                children: [
                                  SizedBox(
                                    width: 16,
                                    height: 16,
                                    child: CircularProgressIndicator(
                                      strokeWidth: 2,
                                      color: Colors.white,
                                    ),
                                  ),
                                  SizedBox(width: 12),
                                  Text('Connecting to server...'),
                                ],
                              ),
                              duration: Duration(seconds: 2),
                            ),
                          );
                          await api.checkHealth();
                        },
                      )),
                ] else if (discovery.isSearching) ...[
                  const ServerListShimmer(),
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
                        onPressed: () => _showQRScanner(context),
                        icon: const Icon(Icons.qr_code_scanner),
                        label: const Text('Scan QR'),
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

  void _showQRScanner(BuildContext context) {
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (context) => Scaffold(
          appBar: AppBar(title: const Text('Scan Server QR Code')),
          body: MobileScanner(
            onDetect: (capture) {
              final List<Barcode> barcodes = capture.barcodes;
              for (final barcode in barcodes) {
                if (barcode.rawValue != null) {
                  final String code = barcode.rawValue!;
                  final discovery = context.read<DiscoveryService>();
                  final api = context.read<ApiService>();
                  discovery.setFromQR(code);
                  if (discovery.hasServer) {
                    api.setServer(discovery.selectedServer!.url);
                    api.checkHealth();
                  }
                  Navigator.pop(context);
                  return;
                }
              }
            },
          ),
        ),
      ),
    );
  }

  void _showManualDialog(BuildContext outerContext) {
    final hostController = TextEditingController();
    final portController = TextEditingController(text: '8443');
    final scaffoldMessenger = ScaffoldMessenger.of(outerContext);

    showDialog(
      context: outerContext,
      builder: (dialogContext) => AlertDialog(
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
            onPressed: () => Navigator.pop(dialogContext),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () async {
              final host = hostController.text.trim();
              final port = int.tryParse(portController.text) ?? 8443;
              if (host.isNotEmpty) {
                final discovery = outerContext.read<DiscoveryService>();
                final api = outerContext.read<ApiService>();
                discovery.setManualServer(host, port);
                api.setServer('https://$host:$port');
                Navigator.pop(dialogContext);
                final connected = await api.checkHealth();
                if (!connected) {
                  scaffoldMessenger.showSnackBar(
                    SnackBar(
                      content: Text(
                        'Connection failed: ${api.error ?? "Unknown error"}',
                      ),
                      backgroundColor: Colors.red.shade700,
                      duration: const Duration(seconds: 4),
                    ),
                  );
                }
              }
            },
            child: const Text('Connect'),
          ),
        ],
      ),
    );
  }
}
