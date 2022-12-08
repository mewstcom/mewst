# typed: strict
# frozen_string_literal: true

class Post < ApplicationRecord
  belongs_to :profile

  sig { returns(String) }
  def inbox_item_score
    T.must(created_at).to_datetime.strftime("%Q")
  end
end
