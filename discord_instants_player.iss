; Script generated by the Inno Script Studio Wizard.
; SEE THE DOCUMENTATION FOR DETAILS ON CREATING INNO SETUP SCRIPT FILES!

[Setup]
; NOTE: The value of AppId uniquely identifies this application.
; Do not use the same AppId value in installers for other applications.
; (To generate a new GUID, click Tools | Generate GUID inside the IDE.)
AppId={{71968401-DB93-438D-9B91-1C18EB9A05D1}
AppName=Discord Instants Player
AppVersion=1.0
;AppVerName=Discord Instants Player 1.0
AppPublisher=github.com/pinheirolucas
DefaultDirName={commonpf}\Discord Instants Player
DefaultGroupName=Discord Instants Player
AllowNoIcons=yes
OutputBaseFilename=setup_discord_instants_player_windows
OutputDir=dist
Compression=lzma
SolidCompression=yes
ChangesEnvironment=yes
PrivilegesRequired=none

[Registry]
Root: HKCU; Subkey: "Environment"; ValueType:string; ValueName: "Path"; ValueData: "{olddata};{app}\ffmpeg\bin"; Flags: preservestringtype; Check: NeedsAddPath('{app}\ffmpeg\bin')

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "bin\discord_instants_player.exe"; DestDir: "{app}"
Source: "bin\ffmpeg.zip"; DestDir: "{tmp}"; Flags: deleteafterinstall
Source: "bin\unzip.exe"; DestDir: "{tmp}"; Flags: deleteafterinstall

[Icons]
Name: "{group}\instants-server"; Filename: "{cmd}"; Parameters: "/c ""{app}\discord_instants_player.exe"""
Name: "{group}\{cm:UninstallProgram,Discord Instants Player}"; Filename: "{uninstallexe}"
Name: "{commondesktop}\instants-server"; Filename: "{cmd}"; Parameters: "/c ""{app}\discord_instants_player.exe"""; Tasks: desktopicon

[Run]
Filename: "{app}\discord_instants_player.exe"; Description: "{cm:LaunchProgram,Discord Instants Player}"; Flags: nowait postinstall skipifsilent
Filename: "{tmp}\unzip.exe"; Parameters: "-oq ""{tmp}\ffmpeg.zip"" -d ""{app}\ffmpeg"""

[UninstallDelete]
Type: filesandordirs; Name: "{app}"

[Code]
function NeedsAddPath(Param: string): boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', OrigPath) then
  begin
    Result := True;
    exit;
  end;
  Result := Pos(';' + ExpandConstant(Param) + ';', ';' + OrigPath + ';') = 0;
end;

var BotSettingsPage: TInputQueryWizardPage;
procedure InitializeWizard;
begin
  BotSettingsPage := CreateInputQueryPage(wpWelcome,
    'Bot settings', 'What are the settings for the bot?',
    'Please specify the bot settings you will use to play the instants.');
  BotSettingsPage.Add('Discord username:', False);
  BotSettingsPage.Add('Bot token:', True);
end;

function GetSettingsFile(): String;
var
  Docs: String;
begin
  Docs := ExpandConstant('{userdocs}');
  Delete(Docs, Pos('Documents', Docs), length('Documents'));
  Result := Docs + '.discord_instants_player.yaml'
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  Saved: Boolean;
begin
  Saved := SaveStringToFile(GetSettingsFile(),
    'discord_instants_player_token: ' + BotSettingsPage.Values[1] + #13#10 +
    'discord_instants_player_owner: ' + BotSettingsPage.Values[0] + #13#10 +
    'discord_instants_player_address: ":9001"' + #13#10,
    True);

  Result := '';
end;

function NextButtonClick(CurPageID: Integer): Boolean;
begin
  { Validate certain pages before allowing the user to proceed }
  if CurPageID = BotSettingsPage.ID then begin
    if BotSettingsPage.Values[0] = '' then begin
      MsgBox('You must enter your discord username.', mbError, MB_OK);
      Result := False;
    end else if BotSettingsPage.Values[1] = '' then begin
      MsgBox('You must enter the bot token.', mbError, MB_OK);
      Result := False;
    end else begin
      Result := True;
    end;
  end else
    Result := True;
end;
