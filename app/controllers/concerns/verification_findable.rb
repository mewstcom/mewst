# typed: strict
# frozen_string_literal: true

module VerificationFindable
  extend T::Sig
  extend ActiveSupport::Concern

  sig { returns(T.untyped) }
  def require_verification_id
    unless session[:verification_id]
      redirect_to root_path
    end
  end
end
