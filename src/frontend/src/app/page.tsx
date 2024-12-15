import UrlShortener from '@/components/UrlShortener'
import ThemeToggle from '@/components/ThemeToggle'

export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-4 sm:p-6 bg-gray-100 dark:bg-gray-900 text-gray-900 dark:text-white transition-colors duration-200">
      <ThemeToggle />
      <h1 className="text-3xl sm:text-4xl md:text-5xl font-bold mb-6 sm:mb-8 text-center text-transparent bg-clip-text bg-gradient-to-r from-blue-600 to-purple-600 dark:from-blue-400 dark:to-purple-400 rounded-lg">
        CHORD URL SHORTENING
      </h1>
      <UrlShortener />
    </main>
  )
}

