/** Display labels only — actual amounts are charged by Razorpay at checkout. Override via Vite env for marketing copy. */

export type PlanCard = {
  id: 'free' | 'starter' | 'team'
  name: string
  priceLine: string
  detail: string
  highlights: string[]
}

const starterPrice =
  (typeof import.meta !== 'undefined' && import.meta.env?.VITE_PRICE_STARTER_LABEL) || 'Paid — set in Razorpay'
const teamPrice =
  (typeof import.meta !== 'undefined' && import.meta.env?.VITE_PRICE_TEAM_LABEL) || 'Paid — set in Razorpay'

export const SUBSCRIPTION_PLANS: PlanCard[] = [
  {
    id: 'free',
    name: 'Free',
    priceLine: '₹0',
    detail: 'Personal vault & org limits for getting started.',
    highlights: ['My Vault: 10 projects, 20 envs', 'Team org: 2 projects, 2 members, 10 envs', 'Unlimited secrets'],
  },
  {
    id: 'starter',
    name: 'Starter',
    priceLine: starterPrice,
    detail: 'Higher limits for individuals and small teams (via Razorpay subscription).',
    highlights: ['Razorpay-managed subscription', 'Limits follow your Envo tier after webhook', 'Upgrade or downgrade in Settings'],
  },
  {
    id: 'team',
    name: 'Team',
    priceLine: teamPrice,
    detail: 'Collaboration-focused tier (via Razorpay subscription).',
    highlights: ['Team workspaces', 'Hosted subscription checkout', 'Manage from Settings after purchase'],
  },
]
