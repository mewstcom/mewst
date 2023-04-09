# typed: strict
# frozen_string_literal: true

class Cards::EntryCardComponent < ApplicationComponent
  sig { params(entry: Entry).void }
  def initialize(entry:)
    @entry = entry
  end

  private

  sig { returns(Entry) }
  attr_reader :entry

  sig { returns(Profile) }
  def profile
    T.must(entry.profile)
  end
end
