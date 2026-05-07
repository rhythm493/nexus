import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../models/message.dart';
import '../services/api_service.dart';
import '../services/cart_service.dart';
import '../services/mode_service.dart';
import '../services/voice_service.dart';
import '../widgets/cart_product_card.dart';
import '../widgets/inline_search.dart';
import '../widgets/agent_chip.dart';
import '../widgets/draggable_divider.dart';
import '../widgets/chat_bubble.dart';
import '../widgets/mode_tabs.dart';
import '../widgets/voice_button.dart';
import '../widgets/llm_settings_section.dart';
import '../widgets/location_settings_section.dart';
import '../widgets/loading_shimmer.dart';
import '../models/cart_full_state.dart';

export '../widgets/loading_shimmer.dart';

class ChatScreen extends StatefulWidget {
  const ChatScreen({super.key});

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final TextEditingController _textController = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  final ScrollController _chatPanelScrollController = ScrollController();
  final List<Message> _messages = [];
  bool _isLoading = false;

  // Grocery mode state
  double _groceryChatRatio = 0.30;
  final List<String> _agentChips = [];
  bool _isAutoScrolling = false;
  Timer? _streamThrottleTimer;
  bool _streamUpdatePending = false;

  @override
  void initState() {
    super.initState();
    // Load hardcoded modes immediately (no network needed)
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<ModeService>().fetchModes('');
    });
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
    final voiceService = context.read<VoiceService>();

    if (apiService.isConnected && apiService.baseUrl != null) {
      // Set server URL for voice model download
      voiceService.setServer(apiService.baseUrl!);
      voiceService.checkModelStatus();

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

    // If already connected (from saved URL), fetch modes
    if (apiService.isConnected && apiService.baseUrl != null) {
      await modeService.fetchModes(apiService.baseUrl!);
      // Set initial mode
      if (modeService.selectedModeId != null) {
        apiService.setMode(modeService.selectedModeId);
      }
      // Set server URL for voice model download + check if model exists
      voiceService.setServer(apiService.baseUrl!);
      voiceService.checkModelStatus();
    }
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


  // --- Grocery mode helpers ---

  bool get _isGroceryMode => context.read<ModeService>().selectedModeId == 'grocery';

  void _handleInlineAdd(String query, String provider) async {
    final api = context.read<ApiService>();
    final cartService = context.read<CartService>();
    if (api.baseUrl == null || api.conversationId == null) return;
    await cartService.addProduct(api.baseUrl!, api.conversationId!, query, provider);
    if (mounted) {
      _addSystemMessage('Added $query');
      _sendHiddenMessage('I added $query from $provider to the cart. Use cart_view to see the updated cart.');
    }
  }

  void _handleCartSwap(String query, String provider) async {
    final api = context.read<ApiService>();
    final cartService = context.read<CartService>();
    if (api.baseUrl == null || api.conversationId == null) return;
    await cartService.swapItem(api.baseUrl!, api.conversationId!, query, provider);
    if (mounted) _addSystemMessage('Swapped $query');
  }

  void _handleCartRemove(String query) async {
    final api = context.read<ApiService>();
    final cartService = context.read<CartService>();
    if (api.baseUrl == null || api.conversationId == null) return;
    await cartService.removeItem(api.baseUrl!, api.conversationId!, query);
    if (mounted) _addSystemMessage('Removed $query');
  }

  void _handleCartClear() {
    context.read<CartService>().clear();
    _addSystemMessage('Cart cleared');
    _sendHiddenMessage('I cleared the cart.');
  }

  void _handleOptimize() => _sendMessage('Optimize my cart');

  void _addSystemMessage(String text) {
    setState(() {
      _messages.add(Message(
        id: DateTime.now().millisecondsSinceEpoch.toString(),
        role: 'system',
        content: text,
        timestamp: DateTime.now(),
      ));
    });
    _scrollToBottom();
  }

  void _sendHiddenMessage(String text) async {
    final api = context.read<ApiService>();
    if (!api.isConnected) return;
    setState(() => _isLoading = true);
    try {
      await for (final event in api.chat(text)) {
        _handleSSEEvent(event);
      }
    } catch (_) {}
    if (mounted) setState(() => _isLoading = false);
  }

  void _handleSSEEvent(SSEEvent event) {
    switch (event.type) {
      case 'cart_update':
        if (mounted) context.read<CartService>().updateFromSSEFull(event.result);
        break;
      case 'cart_optimized':
        if (mounted) context.read<CartService>().updateOptimization(event.result);
        break;
    }
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
      debugPrint('[SSE] type=${event.type}, hasResult=${event.result != null}');
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
        case 'cart_update':
          if (mounted) context.read<CartService>().updateFromSSEFull(event.result);
          break;
        case 'cart_optimized':
          if (mounted) context.read<CartService>().updateOptimization(event.result);
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

          // Mode-specific body
          if (_isGroceryMode) ...[
            Builder(builder: (_) { debugPrint('[BUILD] Rendering GROCERY body, mode=${context.read<ModeService>().selectedModeId}'); return const SizedBox.shrink(); }),
            ..._buildGroceryBody(),
          ] else ...[
            Builder(builder: (_) { debugPrint('[BUILD] Rendering CHAT body, mode=${context.read<ModeService>().selectedModeId}'); return const SizedBox.shrink(); }),
            ..._buildChatBody(),
          ],
        ],
      ),
    );
  }

  List<Widget> _buildChatBody() {
    return [
      // Chat messages
      Expanded(
        child: _messages.isEmpty
            ? const Center(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(Icons.assistant, size: 64, color: Colors.grey),
                    SizedBox(height: 16),
                    Text('Say something or type a message', style: TextStyle(color: Colors.grey)),
                  ],
                ),
              )
            : ListView.builder(
                controller: _scrollController,
                padding: const EdgeInsets.all(16),
                itemCount: _messages.length,
                cacheExtent: 500,
                itemBuilder: (context, index) {
                  final message = _messages[index];
                  return RepaintBoundary(
                    child: ChatBubble(key: ValueKey(message.id), message: message),
                  );
                },
              ),
      ),

      if (_isLoading)
        const Padding(
          padding: EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          child: TypingIndicator(),
        ),

      _buildInputArea(),
    ];
  }

  List<Widget> _buildGroceryBody() {
    final screenHeight = MediaQuery.of(context).size.height;
    final cs = Theme.of(context).colorScheme;
    final collapsed = _groceryChatRatio <= 0.10;

    return [
      // Cart area
      Expanded(
        flex: ((1 - _groceryChatRatio) * 100).round(),
        child: Consumer<CartService>(
          builder: (context, cartService, _) {
            final cart = cartService.fullCart;
            return Column(
              children: [
                InlineSearch(
                  baseUrl: context.read<ApiService>().baseUrl,
                  onAdd: _handleInlineAdd,
                ),
                Expanded(
                  child: cart == null || cart.isEmpty
                      ? Center(
                          child: Column(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              Icon(Icons.shopping_cart_outlined, size: 48, color: cs.onSurface.withValues(alpha: 0.2)),
                              const SizedBox(height: 12),
                              Text('Search above or ask the assistant',
                                  style: TextStyle(color: cs.onSurface.withValues(alpha: 0.4), fontSize: 13)),
                            ],
                          ),
                        )
                      : ListView(
                          padding: const EdgeInsets.symmetric(horizontal: 12),
                          children: [
                            ...cart.items.map((item) => CartProductCard(
                                  item: item,
                                  onSwap: (provider) => _handleCartSwap(item.query, provider),
                                  onRemove: () => _handleCartRemove(item.query),
                                )),
                            ..._agentChips.asMap().entries.map((e) => AgentChip(
                                  text: e.value,
                                  onDismiss: () => setState(() => _agentChips.removeAt(e.key)),
                                )),
                            if (cart.optimization != null)
                              _buildOptimizationBanner(cart),
                          ],
                        ),
                ),
                if (cart != null && !cart.isEmpty) _buildCartFooter(cart),
              ],
            );
          },
        ),
      ),

      // Draggable divider
      GestureDetector(
        onVerticalDragUpdate: (details) {
          setState(() {
            _groceryChatRatio = (_groceryChatRatio - details.primaryDelta! / screenHeight)
                .clamp(0.08, 0.70);
          });
        },
        child: const DraggableDivider(),
      ),

      // Chat panel
      Expanded(
        flex: (_groceryChatRatio * 100).round(),
        child: Column(
          children: [
            if (!collapsed)
              Expanded(
                child: ListView.builder(
                  controller: _chatPanelScrollController,
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                  itemCount: _messages.length,
                  itemBuilder: (context, index) {
                    final msg = _messages[index];
                    if (msg.role == 'system') return _buildSystemBubble(msg.content);
                    final isCompleted = msg.isToolCall &&
                        index + 1 < _messages.length &&
                        _messages[index + 1].isToolResult &&
                        _messages[index + 1].toolName == msg.toolName;
                    if (msg.isToolResult && index > 0 && _messages[index - 1].isToolCall &&
                        _messages[index - 1].toolName == msg.toolName) {
                      return const SizedBox.shrink();
                    }
                    return ChatBubble(key: ValueKey(msg.id), message: msg, completed: isCompleted);
                  },
                ),
              ),
            if (_isLoading && !collapsed)
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
                child: Row(
                  children: [
                    SizedBox(width: 12, height: 12, child: CircularProgressIndicator(strokeWidth: 2, color: cs.primary)),
                    const SizedBox(width: 8),
                    Text('Thinking...', style: TextStyle(fontSize: 11, color: cs.onSurface.withValues(alpha: 0.5))),
                  ],
                ),
              ),
            _buildInputArea(),
          ],
        ),
      ),
    ];
  }

  Widget _buildCartFooter(CartFullState cart) {
    final cs = Theme.of(context).colorScheme;
    final total = cart.cheapestTotal;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      decoration: BoxDecoration(
        color: cs.surfaceContainerLow,
        border: Border(top: BorderSide(color: cs.outline.withValues(alpha: 0.15))),
      ),
      child: Row(
        children: [
          Text('Total: ${CartProduct.formatPaise(total)}',
              style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: cs.onSurface)),
          Text(' \u00b7 ${cart.itemCount} item${cart.itemCount == 1 ? '' : 's'}',
              style: TextStyle(fontSize: 12, color: cs.onSurface.withValues(alpha: 0.5))),
          const Spacer(),
          if (!cart.isOptimized)
            FilledButton.tonal(
              onPressed: _handleOptimize,
              style: FilledButton.styleFrom(
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                minimumSize: Size.zero, tapTargetSize: MaterialTapTargetSize.shrinkWrap,
              ),
              child: const Text('Optimize', style: TextStyle(fontSize: 12)),
            ),
          const SizedBox(width: 8),
          IconButton(onPressed: _handleCartClear, icon: Icon(Icons.delete_outline, size: 18, color: cs.error),
              padding: EdgeInsets.zero, constraints: const BoxConstraints(), tooltip: 'Clear cart'),
        ],
      ),
    );
  }

  Widget _buildOptimizationBanner(CartFullState cart) {
    final opt = cart.optimization!;
    final cs = Theme.of(context).colorScheme;
    if (opt.isSplit) {
      return Container(
        margin: const EdgeInsets.only(top: 4, bottom: 8),
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(color: Colors.green.withValues(alpha: 0.1), borderRadius: BorderRadius.circular(8)),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Icon(Icons.savings_outlined, size: 16, color: Colors.green[400]),
              const SizedBox(width: 6),
              Text('Split saves ${CartProduct.formatPaise(opt.splitSavings)} (${opt.splitSavingsPct.toInt()}%)',
                  style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: Colors.green[400])),
            ]),
            ...opt.splitCarts.map((pc) => Padding(
                  padding: const EdgeInsets.only(left: 22, top: 2),
                  child: Text('${pc.provider}: ${pc.itemCount} items \u2014 ${CartProduct.formatPaise(pc.totalPaise)}',
                      style: TextStyle(fontSize: 11, color: cs.onSurface.withValues(alpha: 0.6))),
                )),
          ],
        ),
      );
    }
    return Container(
      margin: const EdgeInsets.only(top: 4, bottom: 8),
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(color: cs.primary.withValues(alpha: 0.1), borderRadius: BorderRadius.circular(8)),
      child: Row(children: [
        Icon(Icons.check_circle_outline, size: 16, color: cs.primary),
        const SizedBox(width: 6),
        Text('Best: All from ${opt.singleBest.provider} \u2014 ${CartProduct.formatPaise(opt.singleBest.totalPaise)}',
            style: TextStyle(fontSize: 12, fontWeight: FontWeight.w500, color: cs.primary)),
      ]),
    );
  }

  Widget _buildSystemBubble(String text) {
    final cs = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(children: [
        Expanded(child: Divider(color: cs.outline.withValues(alpha: 0.15))),
        Padding(padding: const EdgeInsets.symmetric(horizontal: 8),
            child: Text(text, style: TextStyle(fontSize: 10, color: cs.onSurface.withValues(alpha: 0.4)))),
        Expanded(child: Divider(color: cs.outline.withValues(alpha: 0.15))),
      ]),
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
    );
  }

  @override
  void dispose() {
    _streamThrottleTimer?.cancel();
    final apiService = context.read<ApiService>();
    apiService.removeListener(_onApiServiceChange);
    _textController.dispose();
    _scrollController.dispose();
    _chatPanelScrollController.dispose();
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

              // Language Model section
              Text(
                'Language Model',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 8),
              const LLMSettingsSection(),

              const SizedBox(height: 24),

              // Location section
              const LocationSettingsSection(),

              const SizedBox(height: 24),

              // Server Connection (MOVED TO BOTTOM)
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
                const SizedBox(height: 8),

                // Current server URL
                if (api.baseUrl != null) ...[
                  Text(
                    'Server: ${api.baseUrl}',
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  const SizedBox(height: 16),
                ],

                // Change server button
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

                // Reset to default button (show only if not using default)
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

                // Error message if any
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
    final hostController = TextEditingController(text: api.baseUrl ?? 'https://pocket-assistant-nexus.duckdns.org');
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
                // Ensure URL has proper format
                var finalUrl = url;
                if (!finalUrl.startsWith('http://') && !finalUrl.startsWith('https://')) {
                  finalUrl = 'https://$finalUrl';
                }

                api.setServer(finalUrl);
                outerContext.read<VoiceService>().setServer(finalUrl);
                outerContext.read<VoiceService>().checkModelStatus();
                Navigator.pop(dialogContext);

                // Show loading and try to connect
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
