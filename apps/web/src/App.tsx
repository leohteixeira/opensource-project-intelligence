import { config } from './config';

export function App() {
  return (
    <main>
      <h1>Open Source Project Intelligence</h1>
      <p>
        Fundação da aplicação. A interface consome o backend exclusivamente pelo contrato HTTP
        versionado em <code>{config.apiBaseUrl}/api/v1</code>.
      </p>
    </main>
  );
}
