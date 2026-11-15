-- luadeck_example/demo.lua: an example of making drako profiles with a script.
--
-- It writes small demo profiles into the inventory folder:
--
--   inventory/demo-grid.profile.toml     a 3x3 grid, made by a loop
--   inventory/demo-hello.profile.toml    one folder, made from a list
--
-- Every cell only prints a line of text. Run it from any folder:
--
--   cd ~/.config/drako/luadeck_example
--   drako run-lua demo.lua            both profiles
--   drako run-lua demo.lua --grid     only demo-grid
--   drako run-lua demo.lua --hello    only demo-hello
--
-- Then start drako, press i, and equip them. A file that already exists is
-- left alone. drako does not load Lua: this script only writes TOML files,
-- which drako reads like any other profile. Copy it and make your own.

-- The inventory is the folder beside this script's own folder, wherever the
-- script is run from. arg[0] is the script's path as it was typed.
local here = arg[0]:match("^(.*)[/\\]") or "."
local inventory = here .. "/../inventory"

-- A TOML string: quotes around it, and \ and " escaped.
local function q(s)
  return '"' .. s:gsub('\\', '\\\\'):gsub('"', '\\"') .. '"'
end

-- Writes one profile file into the inventory, unless it is already there.
local function write(name, lines)
  local shown = "inventory/" .. name .. ".profile.toml"
  local path = inventory .. "/" .. name .. ".profile.toml"
  local there = io.open(path, "r")
  if there then
    there:close()
    print("kept     " .. shown .. " (already there)")
    return
  end
  local f, err = io.open(path, "w")
  if not f then
    io.stderr:write("cannot write " .. path .. ": " .. err .. "\n")
    io.stderr:write("This script expects to sit in a folder inside drako's folder (~/.config/drako).\n")
    os.exit(1)
  end
  f:write(table.concat(lines, "\n"), "\n")
  f:close()
  print("wrote    " .. shown)
end

-- Profile 1: a 3x3 grid. Two loops make the nine cells.
local function grid()
  local lines = { "# Made by luadeck_example/demo.lua", "X = 3", "Y = 3" }
  for _, col in ipairs({ "A", "B", "C" }) do
    for row = 0, 2 do
      local address = col .. row
      local text = "you ran cell " .. address
      lines[#lines + 1] = ""
      lines[#lines + 1] = "[[commands]]"
      lines[#lines + 1] = "name = " .. q("Cell " .. address)
      lines[#lines + 1] = "description = " .. q("Prints: " .. text)
      lines[#lines + 1] = "col = " .. q(col)
      lines[#lines + 1] = "row = " .. row
      lines[#lines + 1] = "command = " .. q("echo " .. text)
      lines[#lines + 1] = "auto_close_execution = false" -- wait, so the line can be read
    end
  end
  write("demo-grid", lines)
end

-- Profile 2: one folder. Each entry of the list becomes an item in it.
local function hello()
  local greetings = {
    { "English", "Hello" },
    { "Spanish", "Hola" },
    { "German", "Hallo" },
    { "Japanese", "Konnichiwa" },
  }
  local lines = {
    "# Made by luadeck_example/demo.lua", "X = 1", "Y = 1", "",
    "[[commands]]",
    "name = " .. q("Say hello"),
    "description = " .. q("A folder: Enter opens it, each item prints a greeting."),
    "col = " .. q("A"),
    "row = 0",
    "items = [",
  }
  for _, g in ipairs(greetings) do
    lines[#lines + 1] = "  { name = " .. q(g[1]) .. ", command = " .. q("echo " .. g[2])
      .. ", auto_close_execution = false },"
  end
  lines[#lines + 1] = "]"
  write("demo-hello", lines)
end

-- The flags pick what to make; with none, both are made.
local makers = { ["--grid"] = grid, ["--hello"] = hello }
local picked = {}
for _, flag in ipairs(arg) do
  if not makers[flag] then
    io.stderr:write("unknown flag: " .. flag .. "\n")
    io.stderr:write("usage: drako run-lua demo.lua [--grid] [--hello]\n")
    os.exit(1)
  end
  picked[#picked + 1] = makers[flag]
end
if #picked == 0 then
  picked = { grid, hello }
end
for _, make in ipairs(picked) do
  make()
end
