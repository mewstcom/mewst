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

  sig { returns(T.untyped) }
  def require_succeeded_verification
    @verification = T.let(Verification.succeeded.find_by(id: session[:verification_id]), T.nilable(Verification))

    unless @verification
      redirect_to root_path
    end
  end
end
