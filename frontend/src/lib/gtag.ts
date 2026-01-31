// Google Ads Conversion Tracking

declare global {
  interface Window {
    gtag: (...args: unknown[]) => void;
  }
}

export const GA_ADS_ID = 'AW-778442301';

// Conversion IDs
export const CONVERSION_IDS = {
  SIGN_UP: 'AW-778442301/EhjuCIqUoPAbEL2smPMC',
  PURCHASE: 'AW-778442301/QKNICOGmoPAbEL2smPMC',
};

// Track Sign Up conversion
export function trackSignUp() {
  if (typeof window !== 'undefined' && window.gtag) {
    window.gtag('event', 'conversion', {
      send_to: CONVERSION_IDS.SIGN_UP,
      value: 1.0,
      currency: 'TRY',
    });
    console.log('📊 Sign Up conversion tracked');
  }
}

// Track Purchase conversion
export function trackPurchase(value: number = 1.0, transactionId?: string) {
  if (typeof window !== 'undefined' && window.gtag) {
    window.gtag('event', 'conversion', {
      send_to: CONVERSION_IDS.PURCHASE,
      value: value,
      currency: 'USD',
      transaction_id: transactionId || '',
    });
    console.log('📊 Purchase conversion tracked', { value, transactionId });
  }
}
