import { Link } from 'react-router-dom'
import { Button } from '../components/Button'

export function LandingPage() {
  return (
    <div className="flex min-h-screen flex-col bg-gradient-to-br from-slate-50 to-slate-100">
      <div className="flex flex-1 items-center justify-center px-4 py-12">
        <div className="w-full max-w-2xl text-center">
          <h1 className="text-5xl font-bold text-slate-900">Envo</h1>
          <p className="mt-4 text-xl text-slate-700">
            Seamless and secure way to share environment variables with your team
          </p>
          <p className="mt-6 text-slate-600">
            CLI-first platform for managing secrets across projects and environments. 
            Keep your team in sync without compromising security.
          </p>
          <div className="mt-10">
            <Link to="/login">
              <Button className="px-8 py-3 text-base">Get Started</Button>
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
