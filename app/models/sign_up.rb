# typed: strict
# frozen_string_literal: true

class SignUp
extend T::Sig

  sig { returns(T::Boolean) }
  def self.stopped?
    Rails.configuration.mewst["disabled_to_sign_up"] == "1"
  end
end
