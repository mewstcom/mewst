# typed: strict
# frozen_string_literal: true

class NotifiableType < T::Enum
  enums do
    Follow = new("Follow")
    Stamp = new("Stamp")
  end
end
