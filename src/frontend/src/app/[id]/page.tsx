import { redirect } from 'next/navigation'
import { getLongUrl } from '@/app/actions'

export default async function ShortUrlRedirect({ params }: { params: { id: string } }) {
  const longUrl = await getLongUrl(params.id)

  if (longUrl) {
    redirect(longUrl)
  } else {
    redirect('/')
  }
}

