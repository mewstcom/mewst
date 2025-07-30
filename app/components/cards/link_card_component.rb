# typed: strict
# frozen_string_literal: true

class Cards::LinkCardComponent < ApplicationComponent
  sig { params(link: LinkRecord).void }
  def initialize(link:)
    @link = link
  end

  sig { returns(LinkRecord) }
  attr_reader :link
  private :link
end
