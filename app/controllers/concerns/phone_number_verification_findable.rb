# typed: strict
# frozen_string_literal: true

module PhoneNumberVerificationFindable
  extend T::Sig
  extend ActiveSupport::Concern

  sig { returns(T.untyped) }
  def require_phone_number_verification_id
    unless session[:phone_number_verification_id]
      redirect_to root_path
    end
  end
end
