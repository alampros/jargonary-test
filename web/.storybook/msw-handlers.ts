import { HttpResponse, http } from 'msw'
import { operatorPage } from '../src/test/fixtures'

// The operator console polls these endpoints every 15 seconds.
export const mswHandlers = [
  http.get('/api/v1/scrapes', () => HttpResponse.json(operatorPage().scrapes)),
  http.get('/api/v1/diagnostics', () => HttpResponse.json(operatorPage().diagnostics)),
]
