'use client'

import { useState } from 'react'
import { shortenUrl } from '@/app/actions'
import { motion } from 'framer-motion'
import { CheckCircleIcon, ClipboardIcon } from '@heroicons/react/24/outline'

export default function UrlShortener() {
  const [longUrl, setLongUrl] = useState('')
  const [shortUrl, setShortUrl] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [copied, setCopied] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true)
    const result = await shortenUrl(longUrl);
    setShortUrl(result);
    setIsLoading(false)
  };

  const copyToClipboard = () => {
    navigator.clipboard.writeText(shortUrl).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  return (
    <motion.div 
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
      className="w-full max-w-md px-4 sm:px-0"
    >
      <form onSubmit={handleSubmit} className="bg-white dark:bg-gray-800 shadow-lg rounded-xl px-4 sm:px-8 pt-6 pb-8 mb-4">
        <div className="mb-4">
          <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="url">
            Enter a long URL
          </label>
          <input
            className="shadow appearance-none border rounded-lg w-full py-2 sm:py-3 px-3 sm:px-4 text-gray-700 dark:text-white leading-tight focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white dark:bg-gray-700"
            id="url"
            type="url"
            placeholder="https://example.com/very/long/url"
            value={longUrl}
            onChange={(e) => setLongUrl(e.target.value)}
            required
          />
        </div>
        <div className="flex items-center justify-center">
          <motion.button
            whileTap={{ scale: 0.95 }}
            className="bg-gradient-to-r from-blue-500 to-purple-600 hover:from-blue-600 hover:to-purple-700 text-white font-bold py-2 sm:py-3 px-4 sm:px-6 rounded-lg focus:outline-none focus:shadow-outline w-full transition-colors duration-200"
            type="submit"
            disabled={isLoading}
          >
            {isLoading ? 'Shortening...' : 'Shorten URL'}
          </motion.button>
        </div>
      </form>
      {shortUrl && (
        <motion.div 
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
          className="bg-white dark:bg-gray-800 shadow-lg rounded-xl p-6" 
        >
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-800 dark:text-gray-200">Shortened URL</h3>
            <CheckCircleIcon className="w-6 h-6 text-green-500" />
          </div>
          <div className="bg-gray-100 dark:bg-gray-700 rounded-lg p-4 mb-4">
            <p className="text-blue-600 dark:text-blue-400 break-all">{shortUrl}</p>
          </div>
          <button
            onClick={copyToClipboard}
            className="w-full bg-blue-500 hover:bg-blue-600 text-white font-bold py-2 px-4 rounded-lg transition-colors duration-200 flex items-center justify-center"
          >
            {copied ? (
              <>
                <CheckCircleIcon className="w-5 h-5 mr-2" />
                Copied!
              </>
            ) : (
              <>
                <ClipboardIcon className="w-5 h-5 mr-2" />
                Copy to Clipboard
              </>
            )}
          </button>
        </motion.div>
      )}
    </motion.div>
  )
}

