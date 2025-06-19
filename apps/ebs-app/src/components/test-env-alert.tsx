'use client'

import { Alert, AlertDescription } from "./ui/alert"

export const TestEnvironmentAlert = () => (
  <Alert variant="default" className="bg-orange-700 rounded-none">
    <AlertDescription className="text-sm leading-none text-center">
      This is a test environment. Data will be deleted without prior notice.
    </AlertDescription>
  </Alert>
)