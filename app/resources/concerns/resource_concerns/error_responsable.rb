# typed: strict
# frozen_string_literal: true

module ResourceConcerns::ErrorResponsable
  extend T::Sig
  extend T::Helpers

  interface!

  sig { abstract.returns(V1::ResponseErrorCode) }
  def code
  end

  sig { abstract.returns(String) }
  def message
  end
end
