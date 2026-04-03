import 'dart:async';
import 'package:flutter/material.dart';

class AgentChip extends StatefulWidget {
  final String text;
  final VoidCallback onDismiss;

  const AgentChip({super.key, required this.text, required this.onDismiss});

  @override
  State<AgentChip> createState() => _AgentChipState();
}

class _AgentChipState extends State<AgentChip> with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<Offset> _slideAnimation;
  Timer? _autoDismiss;
  bool _expanded = false;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(duration: const Duration(milliseconds: 300), vsync: this);
    _slideAnimation = Tween<Offset>(begin: const Offset(1, 0), end: Offset.zero)
        .animate(CurvedAnimation(parent: _controller, curve: Curves.easeOut));
    _controller.forward();
    _autoDismiss = Timer(const Duration(seconds: 10), widget.onDismiss);
  }

  @override
  void dispose() {
    _autoDismiss?.cancel();
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    return SlideTransition(
      position: _slideAnimation,
      child: GestureDetector(
        onTap: () {
          setState(() => _expanded = !_expanded);
          _autoDismiss?.cancel();
        },
        child: Container(
          margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
          decoration: BoxDecoration(
            color: cs.primaryContainer.withValues(alpha: 0.3),
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: cs.primary.withValues(alpha: 0.2)),
          ),
          child: Row(
            children: [
              Icon(Icons.lightbulb_outline, size: 14, color: cs.primary),
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  widget.text,
                  style: TextStyle(fontSize: 12, color: cs.onSurface.withValues(alpha: 0.8)),
                  maxLines: _expanded ? 10 : 1,
                  overflow: _expanded ? TextOverflow.visible : TextOverflow.ellipsis,
                ),
              ),
              GestureDetector(
                onTap: widget.onDismiss,
                child: Icon(Icons.close, size: 14, color: cs.onSurface.withValues(alpha: 0.4)),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
