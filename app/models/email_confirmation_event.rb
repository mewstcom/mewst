# typed: strict
# frozen_string_literal: true

class EmailConfirmationEvent < T::Enum
  enums do
    SignUp = new("sign_up")
    PasswordReset = new("password_reset")
  end
end
