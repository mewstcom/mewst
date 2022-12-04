# typed: strict
# frozen_string_literal: true

module Inboxable
  extend T::Sig
  extend T::Helpers

  interface!

  sig { abstract.returns(String) }
  def inbox_key; end
end
