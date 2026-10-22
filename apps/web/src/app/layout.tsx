import type { Metadata, Viewport } from "next";
import { Nunito } from "next/font/google";
import "./globals.css";

const nunito = Nunito({
  variable: "--font-nunito",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Cantinho do Pomar",
  description: "Plante, colha e cuide do seu cantinho.",
};

export const viewport: Viewport = {
  themeColor: "#7cc142",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="pt-BR" className={nunito.variable}>
      <body>{children}</body>
    </html>
  );
}
