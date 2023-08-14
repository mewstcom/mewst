# typed: strict
# frozen_string_literal: true

module Kernel
  extend T::Sig

  sig { returns(T.self_type) }
  def not_nil!
    self
  end
end

class NilClass
  extend T::Sig

  sig { returns(T.noreturn) }
  def not_nil!
    T.must(self)
  end
end
