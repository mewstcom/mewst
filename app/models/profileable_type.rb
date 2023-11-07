# typed: strict
# frozen_string_literal: true

class ProfileableType < T::Enum
  enums do
    User = new("User")
  end
end
