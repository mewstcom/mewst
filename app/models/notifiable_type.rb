# typed: strict
# frozen_string_literal: true

class NotifiableType < T::Enum
  enums do
    Stamp = new("StampRecord")
  end
end
