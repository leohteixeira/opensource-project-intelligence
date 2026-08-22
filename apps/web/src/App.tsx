import { RouterProvider } from 'react-router';

import { router } from './application/router';

export function App() {
  return <RouterProvider router={router} />;
}
