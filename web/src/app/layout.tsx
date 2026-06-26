import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'OPD Panel',
  description: 'Minecraft server control panel',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
