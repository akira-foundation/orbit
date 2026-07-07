import { useEffect, useState } from "react";
import { api } from "../api";
import type { RuntimesConfig, SystemRuntimeStatus } from "../types";
import { RuntimeGroup } from "./RuntimeGroup";

export function RuntimesList() {
  const [cfg, setCfg] = useState<RuntimesConfig | null>(null);
  const [systemNode, setSystemNode] = useState<SystemRuntimeStatus | null>(null);
  const [systemPHP, setSystemPHP] = useState<SystemRuntimeStatus | null>(null);
  const [systemPython, setSystemPython] = useState<SystemRuntimeStatus | null>(
    null,
  );

  useEffect(() => {
    api.runtimesConfig().then(setCfg);
    api.detectSystemNode().then(setSystemNode);
    api.detectSystemPHP().then(setSystemPHP);
    api.detectSystemPython().then(setSystemPython);
  }, []);

  async function saveCfg(next: RuntimesConfig) {
    setCfg(next);
    await api.saveRuntimesConfig(next);
  }

  return (
    <div className="space-y-2.5">
      <RuntimeGroup
        label="Node.js"
        description="Bundled versions used to run project dev commands. Resolved per project from .nvmrc, .node-version, or package.json engines.node."
        system={systemNode}
        preferChecked={!!cfg && cfg.preferSystemNode}
        onPreferChange={(v) =>
          cfg && saveCfg({ ...cfg, preferSystemNode: v })
        }
        list={() => api.listNodeVersions()}
        install={(version) => api.installNodeVersion(version)}
        remove={(version) => api.removeNodeVersion(version)}
      />
      <RuntimeGroup
        label="PHP"
        description="Bundled versions with php-fpm, used to run Laravel projects. Resolved per project from composer.json require.php."
        system={systemPHP}
        preferChecked={!!cfg && cfg.preferSystemPhp}
        onPreferChange={(v) => cfg && saveCfg({ ...cfg, preferSystemPhp: v })}
        list={() => api.listPHPVersions()}
        install={(version) => api.installPHPVersion(version)}
        remove={(version) => api.removePHPVersion(version)}
      />
      <RuntimeGroup
        label="Python"
        description="Bundled CPython used to run Django, FastAPI, and Flask projects. Resolved per project from .python-version or pyproject.toml."
        system={systemPython}
        preferChecked={!!cfg && cfg.preferSystemPython}
        onPreferChange={(v) =>
          cfg && saveCfg({ ...cfg, preferSystemPython: v })
        }
        list={() => api.listPythonVersions()}
        install={(version) => api.installPythonVersion(version)}
        remove={(version) => api.removePythonVersion(version)}
      />
    </div>
  );
}
