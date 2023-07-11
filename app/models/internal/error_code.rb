# typed: strict
# frozen_string_literal: true

class Internal::ErrorCode < T::Enum
  enums do
    InvalidInputData = new("invalid_input_data")
  end
end
