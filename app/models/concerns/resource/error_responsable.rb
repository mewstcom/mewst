# typed: strict
# frozen_string_literal: true

module Resource::ErrorResponsable
  extend T::Sig
  extend T::Helpers

  interface!

  sig { abstract.returns(Latest::ResponseErrorCode) }
  def code
  end

  sig { abstract.returns(String) }
  def message
  end
end
