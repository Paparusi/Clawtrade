export type Locale = 'en' | 'vi' | 'zh' | 'ja' | 'ko' | 'es';

export interface TranslationStrings {
  // Navigation
  dashboard: string;
  aiChat: string;
  portfolio: string;
  positions: string;
  settings: string;

  // Trading
  buy: string;
  sell: string;
  placeOrder: string;
  cancelOrder: string;
  symbol: string;
  price: string;
  quantity: string;
  side: string;

  // Portfolio
  balance: string;
  unrealizedPnl: string;
  todayPnl: string;
  openPositions: string;

  // Status
  connected: string;
  disconnected: string;
  loading: string;
  error: string;

  // AI
  askAboutTrading: string;
  send: string;
  aiAssistant: string;

  // Accessibility
  skipToContent: string;
  openMenu: string;
  closeMenu: string;
}

const en: TranslationStrings = {
  dashboard: 'Dashboard', aiChat: 'AI Chat', portfolio: 'Portfolio',
  positions: 'Positions', settings: 'Settings',
  buy: 'Buy', sell: 'Sell', placeOrder: 'Place Order', cancelOrder: 'Cancel Order',
  symbol: 'Symbol', price: 'Price', quantity: 'Quantity', side: 'Side',
  balance: 'Balance', unrealizedPnl: 'Unrealized PnL', todayPnl: 'Today PnL',
  openPositions: 'Open Positions',
  connected: 'Connected', disconnected: 'Disconnected', loading: 'Loading...', error: 'Error',
  askAboutTrading: 'Ask about trading...', send: 'Send', aiAssistant: 'AI Assistant',
  skipToContent: 'Skip to content', openMenu: 'Open menu', closeMenu: 'Close menu',
};

const vi: TranslationStrings = {
  dashboard: 'Bảng điều khiển', aiChat: 'AI Chat', portfolio: 'Danh mục',
  positions: 'Vị thế', settings: 'Cài đặt',
  buy: 'Mua', sell: 'Bán', placeOrder: 'Đặt lệnh', cancelOrder: 'Hủy lệnh',
  symbol: 'Mã', price: 'Giá', quantity: 'Số lượng', side: 'Hướng',
  balance: 'Số dư', unrealizedPnl: 'Lãi/lỗ chưa thực hiện', todayPnl: 'Lãi/lỗ hôm nay',
  openPositions: 'Vị thế mở',
  connected: 'Đã kết nối', disconnected: 'Mất kết nối', loading: 'Đang tải...', error: 'Lỗi',
  askAboutTrading: 'Hỏi về giao dịch...', send: 'Gửi', aiAssistant: 'Trợ lý AI',
  skipToContent: 'Chuyển đến nội dung', openMenu: 'Mở menu', closeMenu: 'Đóng menu',
};

const zh: TranslationStrings = {
  dashboard: '仪表盘', aiChat: 'AI 聊天', portfolio: '投资组合',
  positions: '持仓', settings: '设置',
  buy: '买入', sell: '卖出', placeOrder: '下单', cancelOrder: '取消订单',
  symbol: '代码', price: '价格', quantity: '数量', side: '方向',
  balance: '余额', unrealizedPnl: '未实现盈亏', todayPnl: '今日盈亏',
  openPositions: '持仓中',
  connected: '已连接', disconnected: '已断开', loading: '加载中...', error: '错误',
  askAboutTrading: '询问交易...', send: '发送', aiAssistant: 'AI 助手',
  skipToContent: '跳转到内容', openMenu: '打开菜单', closeMenu: '关闭菜单',
};

const ja: TranslationStrings = {
  dashboard: 'ダッシュボード', aiChat: 'AIチャット', portfolio: 'ポートフォリオ',
  positions: 'ポジション', settings: '設定',
  buy: '買い', sell: '売り', placeOrder: '注文する', cancelOrder: '注文取消',
  symbol: '銘柄', price: '価格', quantity: '数量', side: '売買',
  balance: '残高', unrealizedPnl: '含み損益', todayPnl: '本日損益',
  openPositions: '保有ポジション',
  connected: '接続済み', disconnected: '切断', loading: '読み込み中...', error: 'エラー',
  askAboutTrading: '取引について質問...', send: '送信', aiAssistant: 'AIアシスタント',
  skipToContent: 'コンテンツへスキップ', openMenu: 'メニューを開く', closeMenu: 'メニューを閉じる',
};

const ko: TranslationStrings = {
  dashboard: '대시보드', aiChat: 'AI 채팅', portfolio: '포트폴리오',
  positions: '포지션', settings: '설정',
  buy: '매수', sell: '매도', placeOrder: '주문하기', cancelOrder: '주문취소',
  symbol: '종목', price: '가격', quantity: '수량', side: '매매',
  balance: '잔고', unrealizedPnl: '미실현 손익', todayPnl: '오늘 손익',
  openPositions: '보유 포지션',
  connected: '연결됨', disconnected: '연결 끊김', loading: '로딩 중...', error: '오류',
  askAboutTrading: '거래에 대해 질문...', send: '보내기', aiAssistant: 'AI 어시스턴트',
  skipToContent: '콘텐츠로 건너뛰기', openMenu: '메뉴 열기', closeMenu: '메뉴 닫기',
};

const es: TranslationStrings = {
  dashboard: 'Panel', aiChat: 'Chat IA', portfolio: 'Cartera',
  positions: 'Posiciones', settings: 'Configuración',
  buy: 'Comprar', sell: 'Vender', placeOrder: 'Colocar orden', cancelOrder: 'Cancelar orden',
  symbol: 'Símbolo', price: 'Precio', quantity: 'Cantidad', side: 'Lado',
  balance: 'Saldo', unrealizedPnl: 'PnL no realizado', todayPnl: 'PnL de hoy',
  openPositions: 'Posiciones abiertas',
  connected: 'Conectado', disconnected: 'Desconectado', loading: 'Cargando...', error: 'Error',
  askAboutTrading: 'Preguntar sobre trading...', send: 'Enviar', aiAssistant: 'Asistente IA',
  skipToContent: 'Ir al contenido', openMenu: 'Abrir menú', closeMenu: 'Cerrar menú',
};

export const translations: Record<Locale, TranslationStrings> = { en, vi, zh, ja, ko, es };

export function getTranslation(locale: Locale): TranslationStrings {
  return translations[locale] || translations.en;
}

export const SUPPORTED_LOCALES: { code: Locale; name: string; nativeName: string }[] = [
  { code: 'en', name: 'English', nativeName: 'English' },
  { code: 'vi', name: 'Vietnamese', nativeName: 'Tiếng Việt' },
  { code: 'zh', name: 'Chinese', nativeName: '中文' },
  { code: 'ja', name: 'Japanese', nativeName: '日本語' },
  { code: 'ko', name: 'Korean', nativeName: '한국어' },
  { code: 'es', name: 'Spanish', nativeName: 'Español' },
];
