import type { Metadata } from "next";
import { DM_Sans, Fraunces } from "next/font/google";
import Script from "next/script";
import "./globals.css";
import { AuthProvider } from "@/lib/AuthContext";
import { ModalProvider } from "@/components/Modal";

const GA_MEASUREMENT_ID = "G-SZW6X77247";

const dmSans = DM_Sans({
  variable: "--font-dm-sans",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700", "800"],
});

const fraunces = Fraunces({
  variable: "--font-fraunces",
  subsets: ["latin"],
  weight: ["600", "700", "800"],
});

export const metadata: Metadata = {
  title: "Makos.ai - AI Worksheet Generator for Teachers",
  description: "Generate engaging and customized worksheets in seconds with Makos.ai, powered by AI. Save time and enhance student learning.",
  keywords: ["AI", "worksheet", "generator", "teachers", "education", "lesson plans", "quiz", "makos.ai"],
  icons: {
    icon: "/favicon.ico",
  },
  metadataBase: new URL("https://makos.ai"),
  openGraph: {
    title: "Makos.ai - AI Worksheet Generator",
    description: "Create professional worksheets in seconds with AI. Multiple question types, 40+ languages, Bloom's Taxonomy support.",
    url: "https://makos.ai",
    siteName: "Makos.ai",
    images: [
      {
        url: "/logo.png",
        width: 512,
        height: 512,
        alt: "Makos.ai - AI Worksheet Generator for Teachers",
      },
    ],
    locale: "en_US",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "Makos.ai - AI Worksheet Generator",
    description: "Create professional worksheets in seconds with AI. Save hours of prep time!",
    images: ["/logo.png"],
  },
  robots: {
    index: true,
    follow: true,
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="light" style={{ colorScheme: 'light' }}>
      <head>
        {/* Google Analytics */}
        <Script
          src={`https://www.googletagmanager.com/gtag/js?id=${GA_MEASUREMENT_ID}`}
          strategy="afterInteractive"
        />
        <Script id="google-analytics" strategy="afterInteractive">
          {`
            window.dataLayer = window.dataLayer || [];
            function gtag(){dataLayer.push(arguments);}
            gtag('js', new Date());
            gtag('config', '${GA_MEASUREMENT_ID}', {
              page_path: window.location.pathname,
            });
          `}
        </Script>
      </head>
      <body
        className={`${dmSans.variable} ${fraunces.variable} antialiased bg-pattern`}
        style={{ fontFamily: "'DM Sans', system-ui, sans-serif" }}
      >
        <AuthProvider>
          <ModalProvider>
            {children}
          </ModalProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
