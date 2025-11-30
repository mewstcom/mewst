# typed: strict
# frozen_string_literal: true

class ProfileAvatarKind < T::Enum
  enums do
    Default = new("default")
    External = new("external")
    Gravatar = new("gravatar")
  end
end
