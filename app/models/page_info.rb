# typed: strict
# frozen_string_literal: true

class PageInfo < T::Struct
  const :has_next_page, T::Boolean
  const :has_previous_page, T::Boolean
end
