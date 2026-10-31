import Foundation
import Testing

@testable import Core

struct PythonAnalysisTests {
    @Test func djangoProjectUsesRunserverAndPip() throws {
        let dir = try TemporaryDirectory()
        try dir.write("#!/usr/bin/env python\n", to: "manage.py")
        try dir.write("Django==5.0\n", to: "requirements.txt")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.runtimeKind == .python)
        #expect(analysis.framework == "django")
        #expect(analysis.devCommand == "python manage.py runserver 127.0.0.1:$PORT")
        #expect(analysis.devPort == 8000)
        #expect(analysis.packageManager == "pip")
        #expect(analysis.pythonVersion != nil)
        #expect(analysis.scripts.isEmpty)
        #expect(analysis.processes.isEmpty)
    }

    @Test func fastAPIProjectWithUvResolvesRequiresPython() throws {
        let dir = try TemporaryDirectory()
        try dir.write(
            "[project]\nname='api'\nrequires-python='3.12'\ndependencies=['fastapi','uvicorn']\n",
            to: "pyproject.toml")
        try dir.write("", to: "uv.lock")
        try dir.write("app = object()\n", to: "main.py")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.framework == "fastapi")
        #expect(analysis.packageManager == "uv")
        #expect(analysis.devCommand == "uvicorn main:app --host 127.0.0.1 --port $PORT")
        #expect(analysis.pythonVersion == "3.12.13")
    }

    @Test func flaskProjectWithPipenv() throws {
        let dir = try TemporaryDirectory()
        try dir.write("Flask>=3.0\n", to: "requirements.txt")
        try dir.write("[packages]\nflask = \"*\"\n", to: "Pipfile")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.framework == "flask")
        #expect(analysis.packageManager == "pipenv")
        #expect(analysis.devCommand == "flask run --host 127.0.0.1 --port $PORT")
        #expect(analysis.devPort == 5000)
    }

    @Test func pythonVersionFileWins() throws {
        let dir = try TemporaryDirectory()
        try dir.write("requests\n", to: "requirements.txt")
        try dir.write("3.11\n", to: ".python-version")

        #expect(try ProjectAnalyzer().analyze(path: dir.path).pythonVersion == "3.11.15")
    }

    @Test func requirementNamesIgnoreCommentsOptionsAndExtras() throws {
        let dir = try TemporaryDirectory()
        try dir.write("# c\n-r base.txt\nfastapi[all]>=0.110 ; python_version>'3'\n", to: "requirements.txt")

        #expect(try ProjectAnalyzer().analyze(path: dir.path).framework == "fastapi")
    }

    @Test func plainScriptUsesItsEntryFile() throws {
        let dir = try TemporaryDirectory()
        try dir.write("requests", to: "requirements.txt")
        try dir.write("print('hi')\n", to: "app.py")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.framework == "python")
        #expect(analysis.devCommand == "python app.py")
        #expect(analysis.devPort == 8000)
    }
}
