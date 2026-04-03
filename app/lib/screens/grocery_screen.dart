import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../models/message.dart';
import '../models/cart_full_state.dart';
import '../services/api_service.dart';
import '../services/cart_service.dart';
import '../widgets/cart_product_card.dart';
import '../widgets/inline_search.dart';
import '../widgets/agent_chip.dart';
import '../widgets/draggable_divider.dart';
import '../widgets/chat_bubble.dart';
import '../widgets/voice_button.dart';
import '../widgets/mode_tabs.dart';

class GroceryScreen extends StatefulWidget {
  const GroceryScreen({super.key});

  @override
  State<GroceryScreen> createState() => _GroceryScreenState();
}

class _GroceryScreenState extends State<GroceryScreen> {
  final List<Message> _messages = [];
  final _textController = TextEditingController();
  final _chatScrollController = ScrollController();
  bool _isLoading = false;
  double _chatRatio = 0.30;
  static const double _minRatio = 0.08;
  static const double _maxRatio = 0.70;

  // Agent advice chips
  final List<String> _chips = [];

  Timer? _streamThrottleTimer;
  bool _streamUpdatePending = false;

  @override
  void dispose() {
    _streamThrottleTimer?.cancel();
    _textController.dispose();
    _chatScrollController.dispose();
    super.dispose();
  }

  // --- Cart actions (direct UI, no LLM) ---

  void _handleInlineAdd(String query, String provider) async {
    final api = context.read<ApiService>();
    final cartService = context.read<CartService>();
    final baseUrl = api.baseUrl;
    final convId = api.conversationId;
    if (baseUrl == null || convId == null) return;

    await cartService.addProduct(baseUrl, convId, query, provider);
    if (mounted) {
      _addSystemMessage('Added $query');
      _sendHiddenMessage(
          'I added $query from $provider to the cart. Use cart_view to see the updated cart.');
    }
  }

  void _handleSwap(String query, String provider) async {
    final api = context.read<ApiService>();
    final cartService = context.read<CartService>();
    final baseUrl = api.baseUrl;
    final convId = api.conversationId;
    if (baseUrl == null || convId == null) return;

    await cartService.swapItem(baseUrl, convId, query, provider);
    if (mounted) {
      _addSystemMessage('Swapped $query to ${_shortProvider(provider)}');
    }
  }

  void _handleRemove(String query) async {
    final api = context.read<ApiService>();
    final cartService = context.read<CartService>();
    final baseUrl = api.baseUrl;
    final convId = api.conversationId;
    if (baseUrl == null || convId == null) return;

    await cartService.removeItem(baseUrl, convId, query);
    if (mounted) {
      _addSystemMessage('Removed $query');
    }
  }

  void _handleClear() {
    context.read<CartService>().clear();
    _addSystemMessage('Cart cleared');
    _sendHiddenMessage('I cleared the cart.');
  }

  void _handleOptimize() {
    _sendMessage('Optimize my cart');
  }

  // --- Message handling ---

  void _addSystemMessage(String text) {
    setState(() {
      _messages.add(Message(
        id: DateTime.now().millisecondsSinceEpoch.toString(),
        role: 'system',
        content: text,
        timestamp: DateTime.now(),
      ));
    });
    _scrollChat();
  }

  void _sendHiddenMessage(String text) async {
    final api = context.read<ApiService>();
    if (!api.isConnected) return;

    // Don't show as user bubble — just send to LLM
    setState(() => _isLoading = true);
    try {
      await for (final event in api.chat(text)) {
        _handleSSEEvent(event);
      }
    } catch (_) {}
    if (mounted) setState(() => _isLoading = false);
  }

  Future<void> _sendMessage(String text) async {
    if (text.trim().isEmpty) return;

    final api = context.read<ApiService>();
    setState(() {
      _messages.add(Message.user(text.trim()));
      _isLoading = true;
    });
    _textController.clear();
    _scrollChat();

    // Expand chat if collapsed
    if (_chatRatio < 0.15) {
      setState(() => _chatRatio = 0.30);
    }

    String assistantResponse = '';
    try {
      await for (final event in api.chat(text.trim())) {
        _handleSSEEvent(event, onText: (t) => assistantResponse = t);
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _messages.add(Message(
            id: DateTime.now().millisecondsSinceEpoch.toString(),
            role: 'error',
            content: 'Connection error: $e',
            timestamp: DateTime.now(),
          ));
        });
      }
    }

    if (mounted) {
      _streamThrottleTimer?.cancel();
      setState(() {
        if (assistantResponse.isNotEmpty) {
          if (_messages.isNotEmpty && _messages.last.isAssistant) {
            _messages.removeLast();
          }
          // Short responses → agent chip instead of chat bubble
          if (assistantResponse.length < 100 &&
              !assistantResponse.contains('\n')) {
            _chips.add(assistantResponse);
          } else {
            _messages.add(Message.assistant(assistantResponse));
          }
        }
        _isLoading = false;
      });
      _scrollChat();
    }
  }

  void _handleSSEEvent(SSEEvent event, {void Function(String)? onText}) {
    switch (event.type) {
      case 'text':
        final text = event.content ?? '';
        onText?.call(text);
        // Throttle UI updates
        if (!_streamUpdatePending) {
          _streamUpdatePending = true;
          _streamThrottleTimer = Timer(const Duration(milliseconds: 100), () {
            if (mounted) {
              setState(() {
                if (_messages.isNotEmpty && _messages.last.isAssistant) {
                  _messages.removeLast();
                }
                _messages.add(Message.assistant(text));
              });
              _scrollChat();
              _streamUpdatePending = false;
            }
          });
        }
        break;
      case 'thinking':
        setState(() => _messages.add(Message.thinking(event.content ?? '')));
        _scrollChat();
        break;
      case 'tool_call':
        setState(() =>
            _messages.add(Message.toolCall(event.name ?? '', event.args)));
        _scrollChat();
        break;
      case 'tool_result':
        setState(() =>
            _messages.add(Message.toolResult(event.name ?? '', event.result)));
        _scrollChat();
        break;
      case 'cart_update':
        if (mounted) {
          context.read<CartService>().updateFromSSEFull(event.result);
        }
        break;
      case 'cart_optimized':
        if (mounted) {
          context.read<CartService>().updateOptimization(event.result);
        }
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
        _scrollChat();
        break;
      case 'done':
        _streamThrottleTimer?.cancel();
        _streamUpdatePending = false;
        break;
    }
  }

  void _scrollChat() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_chatScrollController.hasClients) {
        _chatScrollController.animateTo(
          _chatScrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
      }
    });
  }

  // --- Build ---

  @override
  Widget build(BuildContext context) {
    final api = context.watch<ApiService>();
    final screenHeight = MediaQuery.of(context).size.height;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Grocery'),
        titleTextStyle: TextStyle(
          fontSize: 18,
          fontWeight: FontWeight.w500,
          color: Theme.of(context).colorScheme.onSurface,
        ),
        actions: [
          // Connection indicator
          Padding(
            padding: const EdgeInsets.only(right: 12),
            child: Icon(
              Icons.circle,
              size: 10,
              color: api.isConnected ? Colors.green : Colors.red,
            ),
          ),
        ],
      ),
      body: Column(
        children: [
          // Mode tabs
          const ModeTabs(),

          // Cart area
          Expanded(
            flex: ((1 - _chatRatio) * 100).round(),
            child: _buildCartArea(),
          ),

          // Draggable divider
          GestureDetector(
            onVerticalDragUpdate: (details) {
              setState(() {
                _chatRatio = (_chatRatio - details.primaryDelta! / screenHeight)
                    .clamp(_minRatio, _maxRatio);
              });
            },
            child: const DraggableDivider(),
          ),

          // Chat panel
          Expanded(
            flex: (_chatRatio * 100).round(),
            child: _buildChatPanel(),
          ),
        ],
      ),
    );
  }

  Widget _buildCartArea() {
    return Consumer<CartService>(
      builder: (context, cartService, _) {
        final cart = cartService.fullCart;

        return Column(
          children: [
            // Inline search
            InlineSearch(
              baseUrl: context.read<ApiService>().baseUrl,
              onAdd: _handleInlineAdd,
            ),

            // Cart items
            Expanded(
              child: cart == null || cart.isEmpty
                  ? _emptyCart()
                  : ListView(
                      padding: const EdgeInsets.symmetric(horizontal: 12),
                      children: [
                        ...cart.items.map((item) => CartProductCard(
                              item: item,
                              onSwap: (provider) =>
                                  _handleSwap(item.query, provider),
                              onRemove: () => _handleRemove(item.query),
                            )),
                        // Agent chips
                        ..._chips.asMap().entries.map((e) => AgentChip(
                              text: e.value,
                              onDismiss: () =>
                                  setState(() => _chips.removeAt(e.key)),
                            )),
                        // Optimization banner
                        if (cart.optimization != null)
                          Padding(
                            padding: const EdgeInsets.only(top: 4, bottom: 8),
                            child: _optimizationBanner(cart),
                          ),
                      ],
                    ),
            ),

            // Total footer
            if (cart != null && !cart.isEmpty) _totalFooter(cart),
          ],
        );
      },
    );
  }

  Widget _emptyCart() {
    final cs = Theme.of(context).colorScheme;
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.shopping_cart_outlined,
              size: 48, color: cs.onSurface.withValues(alpha: 0.2)),
          const SizedBox(height: 12),
          Text('Search above or ask the assistant',
              style: TextStyle(
                  color: cs.onSurface.withValues(alpha: 0.4), fontSize: 13)),
        ],
      ),
    );
  }

  Widget _totalFooter(CartFullState cart) {
    final cs = Theme.of(context).colorScheme;
    final total = cart.cheapestTotal;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      decoration: BoxDecoration(
        color: cs.surfaceContainerLow,
        border:
            Border(top: BorderSide(color: cs.outline.withValues(alpha: 0.15))),
      ),
      child: Row(
        children: [
          Text(
            'Total: ${CartProduct.formatPaise(total)}',
            style: TextStyle(
                fontSize: 14, fontWeight: FontWeight.w600, color: cs.onSurface),
          ),
          Text(
            ' \u00b7 ${cart.itemCount} item${cart.itemCount == 1 ? '' : 's'}',
            style: TextStyle(
                fontSize: 12, color: cs.onSurface.withValues(alpha: 0.5)),
          ),
          const Spacer(),
          if (!cart.isOptimized)
            FilledButton.tonal(
              onPressed: _handleOptimize,
              style: FilledButton.styleFrom(
                padding:
                    const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                minimumSize: Size.zero,
                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
              ),
              child: const Text('Optimize', style: TextStyle(fontSize: 12)),
            ),
          const SizedBox(width: 8),
          IconButton(
            onPressed: _handleClear,
            icon: Icon(Icons.delete_outline, size: 18, color: cs.error),
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(),
            tooltip: 'Clear cart',
          ),
        ],
      ),
    );
  }

  Widget _optimizationBanner(CartFullState cart) {
    final opt = cart.optimization!;
    final cs = Theme.of(context).colorScheme;

    if (opt.isSplit) {
      return Container(
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(
          color: Colors.green.withValues(alpha: 0.1),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.savings_outlined,
                    size: 16, color: Colors.green[400]),
                const SizedBox(width: 6),
                Text(
                  'Split saves ${CartProduct.formatPaise(opt.splitSavings)} (${opt.splitSavingsPct.toInt()}%)',
                  style: TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                      color: Colors.green[400]),
                ),
              ],
            ),
            ...opt.splitCarts.map((pc) => Padding(
                  padding: const EdgeInsets.only(left: 22, top: 2),
                  child: Text(
                    '${_shortProvider(pc.provider)}: ${pc.itemCount} items \u2014 ${CartProduct.formatPaise(pc.totalPaise)}',
                    style: TextStyle(
                        fontSize: 11,
                        color: cs.onSurface.withValues(alpha: 0.6)),
                  ),
                )),
          ],
        ),
      );
    }

    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: cs.primary.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          Icon(Icons.check_circle_outline, size: 16, color: cs.primary),
          const SizedBox(width: 6),
          Text(
            'Best: All from ${_shortProvider(opt.singleBest.provider)} \u2014 ${CartProduct.formatPaise(opt.singleBest.totalPaise)}',
            style: TextStyle(
                fontSize: 12, fontWeight: FontWeight.w500, color: cs.primary),
          ),
        ],
      ),
    );
  }

  Widget _buildChatPanel() {
    final cs = Theme.of(context).colorScheme;
    final collapsed = _chatRatio <= _minRatio + 0.02;

    return Column(
      children: [
        // Chat messages (hidden when collapsed)
        if (!collapsed)
          Expanded(
            child: ListView.builder(
              controller: _chatScrollController,
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
              itemCount: _messages.length,
              itemBuilder: (context, index) {
                final msg = _messages[index];
                if (msg.role == 'system') return _systemBubble(msg.content);

                // Merge tool_call + tool_result
                final isCompleted = msg.isToolCall &&
                    index + 1 < _messages.length &&
                    _messages[index + 1].isToolResult &&
                    _messages[index + 1].toolName == msg.toolName;

                if (msg.isToolResult &&
                    index > 0 &&
                    _messages[index - 1].isToolCall &&
                    _messages[index - 1].toolName == msg.toolName) {
                  return const SizedBox
                      .shrink(); // Hidden — merged into tool_call above
                }

                return ChatBubble(
                  key: ValueKey(msg.id),
                  message: msg,
                  completed: isCompleted,
                );
              },
            ),
          ),

        // Loading indicator
        if (_isLoading && !collapsed)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
            child: Row(
              children: [
                SizedBox(
                    width: 12,
                    height: 12,
                    child: CircularProgressIndicator(
                        strokeWidth: 2, color: cs.primary)),
                const SizedBox(width: 8),
                Text('Thinking...',
                    style: TextStyle(
                        fontSize: 11,
                        color: cs.onSurface.withValues(alpha: 0.5))),
              ],
            ),
          ),

        // Input bar (always visible)
        Container(
          padding: const EdgeInsets.all(6),
          decoration: BoxDecoration(
            color: cs.surface,
            border: Border(
                top: BorderSide(color: cs.outline.withValues(alpha: 0.1))),
          ),
          child: SafeArea(
            child: Row(
              children: [
                VoiceButton(onResult: _sendMessage),
                const SizedBox(width: 6),
                Expanded(
                  child: TextField(
                    controller: _textController,
                    decoration: InputDecoration(
                      hintText: 'Ask the assistant...',
                      hintStyle: TextStyle(
                          fontSize: 13,
                          color: cs.onSurface.withValues(alpha: 0.4)),
                      border: const OutlineInputBorder(),
                      contentPadding: const EdgeInsets.symmetric(
                          horizontal: 12, vertical: 10),
                      isDense: true,
                    ),
                    style: const TextStyle(fontSize: 13),
                    textInputAction: TextInputAction.send,
                    onSubmitted: _sendMessage,
                  ),
                ),
                const SizedBox(width: 6),
                IconButton.filled(
                  onPressed: () => _sendMessage(_textController.text),
                  icon: const Icon(Icons.send, size: 18),
                  padding: const EdgeInsets.all(8),
                  constraints: const BoxConstraints(),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _systemBubble(String text) {
    final cs = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          Expanded(child: Divider(color: cs.outline.withValues(alpha: 0.15))),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: Text(text,
                style: TextStyle(
                    fontSize: 10, color: cs.onSurface.withValues(alpha: 0.4))),
          ),
          Expanded(child: Divider(color: cs.outline.withValues(alpha: 0.15))),
        ],
      ),
    );
  }

  String _shortProvider(String p) {
    switch (p) {
      case 'blinkit':
        return 'Blinkit';
      case 'instamart':
        return 'Instamart';
      case 'zepto':
        return 'Zepto';
      default:
        return p;
    }
  }
}
