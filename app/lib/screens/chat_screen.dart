import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../services/api_service.dart';
import '../services/cart_service.dart';
import '../services/model_cache_service.dart';
import '../services/mode_service.dart';
import '../services/voice_service.dart';
import '../providers/conversation_provider.dart';
import '../widgets/conversation_stream.dart';
import '../widgets/agent_status_bar.dart';
import '../widgets/context_bar.dart';
import '../widgets/quick_actions.dart';
import '../widgets/mode_tabs.dart';
import '../widgets/voice_button.dart';
import '../widgets/llm_settings_section.dart';
import '../widgets/location_settings_section.dart';
import '../widgets/model_info_bar.dart';

export '../widgets/loading_shimmer.dart';

class ChatScreen extends StatefulWidget {
  const ChatScreen({super.key});

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final TextEditingController _textController = TextEditingController();
  final ScrollController _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<ModeService>().fetchModes('');
      context.read<ConversationProvider>().setServices(
        apiService: context.read<ApiService>(),
        cartService: context.read<CartService>(),
      );
    });
    _initialize();

    WidgetsBinding.instance.addPostFrameCallback((_) {
      final apiService = context.read<ApiService>();
      apiService.addListener(_onApiServiceChange);
    });
  }

  void _onApiServiceChange() async {
    final apiService = context.read<ApiService>();
    final modeService = context.read<ModeService>();
    final voiceService = context.read<VoiceService>();
    final modelCache = context.read<ModelCacheService>();

    if (apiService.isConnected && apiService.baseUrl != null) {
      voiceService.setServer(apiService.baseUrl!);
      voiceService.checkModelStatus();

      if (modeService.availableModes.isEmpty && !modeService.isLoading) {
        debugPrint('Fetching modes from ${apiService.baseUrl}');
        try {
          await modeService.fetchModes(apiService.baseUrl!);
        } catch (e) {
          debugPrint('Failed to fetch modes: $e');
        }
      }
      if (modeService.selectedModeId != null &&
          apiService.selectedMode != modeService.selectedModeId) {
        apiService.setMode(modeService.selectedModeId);
      }

      modelCache.load(apiService.baseUrl!);
    }
  }

  Future<void> _initialize() async {
    final voiceService = context.read<VoiceService>();
    final apiService = context.read<ApiService>();
    final modeService = context.read<ModeService>();
    final modelCache = context.read<ModelCacheService>();

    await voiceService.initialize();

    if (apiService.isConnected && apiService.baseUrl != null) {
      await modeService.fetchModes(apiService.baseUrl!);
      if (modeService.selectedModeId != null) {
        apiService.setMode(modeService.selectedModeId);
      }
      voiceService.setServer(apiService.baseUrl!);
      voiceService.checkModelStatus();

      unawaited(modelCache.load(apiService.baseUrl!));
    }
  }

  void _showSettingsSheet() {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (context) => const _SettingsSheet(),
    );
  }

  void _sendMessage(String text) {
    context.read<ConversationProvider>().sendMessage(text, context.read<ApiService>());
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
              context.read<ConversationProvider>().newConversation();
              api.newConversation();
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
          const ModeTabs(),
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
                    const Expanded(child: Text('Not connected to server')),
                    TextButton(
                      onPressed: _showSettingsSheet,
                      child: const Text('Connect'),
                    ),
                  ],
                ),
              );
            },
          ),
          Expanded(child: _buildChatBody()),
          const AgentStatusBar(),
          const ContextBar(),
          const QuickActions(),
          _buildInputArea(),
        ],
      ),
    );
  }

  Widget _buildChatBody() {
    return Consumer<ConversationProvider>(
      builder: (context, provider, _) {
        return ConversationStream(
          messages: provider.messages,
          isLoading: provider.isLoading,
          onAction: (action, args) {
            provider.handleAction(action, args);
          },
          onRetry: (messageId) {
            provider.sendMessage('Continue from where you left off', context.read<ApiService>());
          },
          scrollController: _scrollController,
        );
      },
    );
  }

  Widget _buildInputArea() {
    return Container(
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
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const ModelInfoBar(),
          Row(
            children: [
              VoiceButton(
                onResult: (text) {
                  _textController.text = text;
                  _sendMessage(text);
                },
              ),
              const SizedBox(width: 8),
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
              IconButton.filled(
                onPressed: () => _sendMessage(_textController.text),
                icon: const Icon(Icons.send),
              ),
            ],
          ),
        ],
      ),
    );
  }

  @override
  void dispose() {
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
              Text(
                'Settings',
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              const SizedBox(height: 24),
              Text(
                'Language Model',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 8),
              const LLMSettingsSection(),
              const SizedBox(height: 24),
              const LocationSettingsSection(),
              const SizedBox(height: 24),
              Text(
                'Server Connection',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 8),
              const _ServerSection(),
            ],
          ),
        );
      },
    );
  }
}

class _ServerSection extends StatelessWidget {
  const _ServerSection();

  @override
  Widget build(BuildContext context) {
    return Consumer<ApiService>(
      builder: (context, api, _) {
        return Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
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
                const SizedBox(height: 8),
                if (api.baseUrl != null) ...[
                  Text(
                    'Server: ${api.baseUrl}',
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  const SizedBox(height: 16),
                ],
                Row(
                  children: [
                    Expanded(
                      child: OutlinedButton.icon(
                        onPressed: () => _showManualDialog(context),
                        icon: const Icon(Icons.edit),
                        label: const Text('Change Server'),
                      ),
                    ),
                  ],
                ),
                if (api.baseUrl != 'https://pocket-assistant-nexus.duckdns.org') ...[
                  const SizedBox(height: 8),
                  Row(
                    children: [
                      Expanded(
                        child: OutlinedButton.icon(
                          onPressed: () async {
                            await api.resetToDefault();
                            if (context.mounted) {
                              context.read<VoiceService>().setServer(api.baseUrl!);
                              context.read<VoiceService>().checkModelStatus();
                            }
                          },
                          icon: const Icon(Icons.refresh),
                          label: const Text('Reset to Default'),
                        ),
                      ),
                    ],
                  ),
                ],
                if (api.error != null) ...[
                  const SizedBox(height: 8),
                  Text(
                    api.error!,
                    style: TextStyle(color: Colors.red[400], fontSize: 12),
                  ),
                ],
              ],
            ),
          ),
        );
      },
    );
  }

  void _showManualDialog(BuildContext outerContext) {
    final api = outerContext.read<ApiService>();
    final hostController = TextEditingController(
      text: api.baseUrl ?? 'https://pocket-assistant-nexus.duckdns.org',
    );
    final scaffoldMessenger = ScaffoldMessenger.of(outerContext);

    showDialog(
      context: outerContext,
      builder: (dialogContext) => AlertDialog(
        title: const Text('Change Server'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: hostController,
              decoration: const InputDecoration(
                labelText: 'Server URL',
                hintText: 'https://example.com',
              ),
              keyboardType: TextInputType.url,
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
              final url = hostController.text.trim();
              if (url.isNotEmpty) {
                var finalUrl = url;
                if (!finalUrl.startsWith('http://') && !finalUrl.startsWith('https://')) {
                  finalUrl = 'https://$finalUrl';
                }

                api.setServer(finalUrl);
                outerContext.read<VoiceService>().setServer(finalUrl);
                outerContext.read<VoiceService>().checkModelStatus();
                Navigator.pop(dialogContext);

                final connected = await api.checkHealth();
                if (!connected && outerContext.mounted) {
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
