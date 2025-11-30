# typed: strict
# frozen_string_literal: true

class SignUpStopper
  extend T::Sig

  sig { returns(T::Boolean) }
  def self.enabled?
    Rails.configuration.mewst["sign_up_stopper_enabled"] == true
  end
end
