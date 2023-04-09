# typed: strict
# frozen_string_literal: true

class Dropdowns::EntryOptionsDropdownComponent < ApplicationComponent
  sig { params(entry: Entry).void }
  def initialize(entry:)
    @entry = entry
  end
end
