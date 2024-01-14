# typed: strict
# frozen_string_literal: true

module ModelConcerns::TimelineOwnable
  extend T::Sig
  extend T::Helpers

  interface!

  sig { abstract.returns(String) }
  def timeline_key
  end
end
