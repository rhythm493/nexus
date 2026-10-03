import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'core/content_block.dart';
import 'core/widget_registry.dart';
import 'providers/conversation_provider.dart';
import 'services/api_service.dart';
import 'services/cart_service.dart';
import 'services/location_service.dart';
import 'services/mode_service.dart';
import 'services/model_cache_service.dart';
import 'services/settings_service.dart';
import 'services/voice_service.dart';
import 'screens/chat_screen.dart';

// Component imports
import 'components/display/markdown_component.dart';
import 'components/display/card_component.dart';
import 'components/display/metric_component.dart';
import 'components/display/table_component.dart';
import 'components/display/image_component.dart';
import 'components/display/code_component.dart';
import 'components/display/list_component.dart';
import 'components/display/grid_component.dart';
import 'components/display/progress_component.dart';
import 'components/display/badge_component.dart';
import 'components/display/chip_component.dart';
import 'components/display/divider_component.dart';
import 'components/display/search_results_component.dart';
import 'components/display/text_component.dart';
import 'components/layout/row_component.dart';
import 'components/layout/column_component.dart';
import 'components/layout/expanded_component.dart';
import 'components/layout/wrap_component.dart';
import 'components/layout/stack_component.dart';
import 'components/layout/scroll_component.dart';
import 'components/layout/container_component.dart';
import 'components/layout/spacer_component.dart';
import 'components/interactive/button_component.dart';
import 'components/interactive/form_component.dart';
import 'components/interactive/toggle_component.dart';
import 'components/interactive/slider_component.dart';
import 'services/auth_service.dart';

void main() {
  _registerComponents();
  // Deliberately not awaited: loading the Google plugin and restoring the
  // account can take a moment, and the UI should not wait on it. Anything that
  // needs a session calls ensureSession(), which awaits init() itself.
  AuthService.instance.init();
  runApp(const NexusApp());
}

void _registerComponents() {
  // Display components
  registerMarkdownComponent();
  registerCardComponent();
  registerMetricComponent();
  registerTableComponent();
  registerImageComponent();
  registerCodeComponent();
  registerListComponent();
  registerGridComponent();
  registerProgressComponent();
  registerBadgeComponent();
  registerChipComponent();
  registerDividerComponent();
  registerTextComponent();

  // Tool result components
  SearchResultsComponent.register();

  // Layout components
  WidgetRegistry.register(ComponentType.row, RowComponent.build);
  WidgetRegistry.register(ComponentType.column, ColumnComponent.build);
  WidgetRegistry.register(ComponentType.expanded, ExpandedComponent.build);
  WidgetRegistry.register(ComponentType.wrap, WrapComponent.build);
  WidgetRegistry.register(ComponentType.stack, StackComponent.build);
  WidgetRegistry.register(ComponentType.scroll, ScrollComponent.build);
  WidgetRegistry.register(ComponentType.container, ContainerComponent.build);
  WidgetRegistry.register(ComponentType.spacer, SpacerComponent.build);

  // Interactive components
  registerButtonComponent();
  registerFormComponent();
  registerToggleComponent();
  registerSliderComponent();

  // Stub registrations for components not yet created
  WidgetRegistry.register(ComponentType.chart, (block, context, {onAction}) {
    return Text('[chart] ${block.data['type']}');
  });
  WidgetRegistry.register(ComponentType.dropdown, (block, context, {onAction}) {
    return Text('[dropdown] ${block.data['label']}');
  });
  WidgetRegistry.register(ComponentType.link, (block, context, {onAction}) {
    return Text('[link] ${block.data['text']}');
  });
}

class NexusApp extends StatelessWidget {
  const NexusApp({super.key});

  @override
  Widget build(BuildContext context) {
    final modelCache = ModelCacheService();

    return MultiProvider(
      providers: [
        ChangeNotifierProvider(create: (_) => ApiService()),
        ChangeNotifierProvider(create: (_) => VoiceService()),
        ChangeNotifierProvider(create: (_) => modelCache),
        ChangeNotifierProvider(create: (_) => SettingsService(modelCache: modelCache)),
        ChangeNotifierProvider(create: (_) => ModeService()),
        ChangeNotifierProvider(create: (_) => LocationService()),
        ChangeNotifierProvider(create: (_) => CartService()),
        // Singleton: ApiService and the other services reach it directly to stamp
        // Authorization headers, so it must be the same instance everywhere.
        // .value rather than create, so we do not construct a second one.
        ChangeNotifierProvider<AuthService>.value(value: AuthService.instance),
        ChangeNotifierProvider(
          create: (context) => ConversationProvider()
            ..setServices(
              apiService: context.read<ApiService>(),
              cartService: context.read<CartService>(),
            ),
        ),
      ],
      child: MaterialApp(
        title: 'Nexus',
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
